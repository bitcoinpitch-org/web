package antispam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/models"

	"github.com/google/uuid"
)

// Service handles antispam operations
type Service struct {
	repo          *database.Repository
	configService *config.Service
}

// NewService creates a new antispam service
func NewService(repo *database.Repository, configService *config.Service) *Service {
	return &Service{
		repo:          repo,
		configService: configService,
	}
}

// CheckPitchCreation checks if a user can create a new pitch
func (s *Service) CheckPitchCreation(ctx context.Context, userID *uuid.UUID, content string, ipAddress net.IP, userAgent string) (*models.AntiSpamCheck, error) {
	result := models.NewAntiSpamCheck(true)

	// Check if user is authenticated (required for detailed checks)
	if userID == nil {
		// For anonymous users, apply basic IP-based limits
		return s.checkAnonymousLimits(ctx, ipAddress, models.ActivityTypePitchCreate)
	}

	// Check daily pitch limits
	if err := s.checkDailyPitchLimits(ctx, *userID, result); err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Check cooldown periods
	if err := s.checkCooldownPeriods(ctx, *userID, models.ActivityTypePitchCreate, result); err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Check content-based restrictions
	if err := s.checkContentRestrictions(ctx, *userID, content, result); err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Check for rapid actions and apply progressive penalties if needed
	if err := s.checkRapidActions(ctx, *userID, models.ActivityTypePitchCreate, result); err != nil {
		return nil, err
	}

	return result, nil
}

// CheckPitchEdit checks if a user can edit a pitch
func (s *Service) CheckPitchEdit(ctx context.Context, userID uuid.UUID, pitchID uuid.UUID, content string, ipAddress net.IP, userAgent string) (*models.AntiSpamCheck, error) {
	result := models.NewAntiSpamCheck(true)

	// Check cooldown periods for edits
	if err := s.checkCooldownPeriods(ctx, userID, models.ActivityTypePitchEdit, result); err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Check content-based restrictions
	if err := s.checkContentRestrictions(ctx, userID, content, result); err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Check for rapid actions
	if err := s.checkRapidActions(ctx, userID, models.ActivityTypePitchEdit, result); err != nil {
		return nil, err
	}

	return result, nil
}

// CheckVote checks if a user can vote
func (s *Service) CheckVote(ctx context.Context, userID *uuid.UUID, pitchID uuid.UUID, ipAddress net.IP, userAgent string) (*models.AntiSpamCheck, error) {
	result := models.NewAntiSpamCheck(true)

	if userID == nil {
		// Anonymous users can vote but with stricter limits
		return s.checkAnonymousLimits(ctx, ipAddress, models.ActivityTypeVote)
	}

	// Check cooldown for votes (shorter than pitches)
	if err := s.checkCooldownPeriods(ctx, *userID, models.ActivityTypeVote, result); err != nil {
		return nil, err
	}

	return result, nil
}

// RecordActivity records a user activity for tracking
func (s *Service) RecordActivity(ctx context.Context, userID *uuid.UUID, actionType models.ActivityType, targetID *uuid.UUID, ipAddress net.IP, userAgent string, metadata map[string]interface{}) error {
	activity := models.NewUserActivity(userID, actionType, targetID, &ipAddress, &userAgent)

	// Add metadata
	for key, value := range metadata {
		activity.SetMetadata(key, value)
	}

	return s.repo.CreateUserActivity(ctx, activity)
}

// RecordContentHash records a content hash for duplicate detection
func (s *Service) RecordContentHash(ctx context.Context, userID uuid.UUID, content string, pitchID uuid.UUID) error {
	hash := s.generateContentHash(content)
	contentHash := models.NewContentHash(userID, hash, content, pitchID)
	return s.repo.CreateContentHash(ctx, contentHash)
}

// checkDailyPitchLimits checks if user has exceeded daily pitch limits
func (s *Service) checkDailyPitchLimits(ctx context.Context, userID uuid.UUID, result *models.AntiSpamCheck) error {
	maxPitchesPerDay := s.configService.GetInt(ctx, "users.max_pitches_per_day", 10)

	// Check active penalties that might modify the limit
	penalties, err := s.repo.GetActivePenaltiesForUser(ctx, userID)
	if err != nil {
		return err
	}

	// Apply penalty multipliers (penalties reduce the limit)
	effectiveLimit := float64(maxPitchesPerDay)
	for _, penalty := range penalties {
		if penalty.PenaltyType == models.PenaltyTypeRateLimit {
			effectiveLimit = effectiveLimit / penalty.Multiplier
		}
	}

	// Count pitches created today
	todayStart := time.Now().Truncate(24 * time.Hour)
	count, err := s.repo.CountUserActivitiesSince(ctx, userID, models.ActivityTypePitchCreate, todayStart)
	if err != nil {
		return err
	}

	if count >= int(effectiveLimit) {
		result.Allowed = false
		result.SetReason(fmt.Sprintf("Daily pitch limit exceeded (%d/%d)", count, int(effectiveLimit)))

		// Calculate retry after (time until next day)
		tomorrow := todayStart.Add(24 * time.Hour)
		retryAfter := time.Until(tomorrow)
		result.SetRetryAfter(retryAfter)

		// Add penalty info if applicable
		for _, penalty := range penalties {
			if penalty.PenaltyType == models.PenaltyTypeRateLimit {
				result.AddPenalty(penalty)
			}
		}
	}

	return nil
}

// checkCooldownPeriods checks if enough time has passed since last action
func (s *Service) checkCooldownPeriods(ctx context.Context, userID uuid.UUID, actionType models.ActivityType, result *models.AntiSpamCheck) error {
	var cooldownSeconds int

	switch actionType {
	case models.ActivityTypePitchCreate:
		cooldownSeconds = s.configService.GetInt(ctx, "antispam.pitch_create_cooldown_seconds", 60)
	case models.ActivityTypePitchEdit:
		cooldownSeconds = s.configService.GetInt(ctx, "antispam.pitch_edit_cooldown_seconds", 30)
	case models.ActivityTypeVote:
		cooldownSeconds = s.configService.GetInt(ctx, "antispam.vote_cooldown_seconds", 2)
	default:
		return nil // No cooldown for other actions
	}

	// Check active penalties that might modify the cooldown
	penalties, err := s.repo.GetActivePenaltiesForUser(ctx, userID)
	if err != nil {
		return err
	}

	// Apply penalty multipliers (penalties increase the cooldown)
	effectiveCooldown := float64(cooldownSeconds)
	for _, penalty := range penalties {
		if penalty.PenaltyType == models.PenaltyTypeCooldown {
			effectiveCooldown = effectiveCooldown * penalty.Multiplier
		}
	}

	// Get last activity of this type
	lastActivity, err := s.repo.GetLastUserActivity(ctx, userID, actionType)
	if err != nil {
		// If no previous activity, allow the action
		return nil
	}

	if lastActivity != nil {
		timeSince := time.Since(lastActivity.CreatedAt)
		requiredCooldown := time.Duration(effectiveCooldown) * time.Second

		if timeSince < requiredCooldown {
			result.Allowed = false
			result.SetReason(fmt.Sprintf("Cooldown period not met for %s", actionType))

			retryAfter := requiredCooldown - timeSince
			result.SetRetryAfter(retryAfter)

			// Add penalty info if applicable
			for _, penalty := range penalties {
				if penalty.PenaltyType == models.PenaltyTypeCooldown {
					result.AddPenalty(penalty)
				}
			}
		}
	}

	return nil
}

// checkContentRestrictions checks for content-based spam indicators
func (s *Service) checkContentRestrictions(ctx context.Context, userID uuid.UUID, content string, result *models.AntiSpamCheck) error {
	// Check minimum/maximum length
	minLength := s.configService.GetInt(ctx, "antispam.min_pitch_length", 3)
	maxLength := s.configService.GetInt(ctx, "antispam.max_pitch_length", 2048)

	if len(content) < minLength {
		result.Allowed = false
		result.SetReason(fmt.Sprintf("Content too short (minimum %d characters)", minLength))
		return nil
	}

	if len(content) > maxLength {
		result.Allowed = false
		result.SetReason(fmt.Sprintf("Content too long (maximum %d characters)", maxLength))
		return nil
	}

	// Check blacklisted phrases
	blacklistedPhrases := s.configService.GetStringSlice(ctx, "antispam.blacklisted_phrases")
	contentLower := strings.ToLower(content)

	for _, phrase := range blacklistedPhrases {
		if phrase != "" && strings.Contains(contentLower, strings.ToLower(phrase)) {
			result.Allowed = false
			result.SetReason("Content contains prohibited phrases")
			return nil
		}
	}

	// Check for duplicate content
	contentHash := s.generateContentHash(content)
	similarContent, err := s.repo.GetContentHashesByHash(ctx, contentHash)
	if err != nil {
		return err
	}

	// Check if user has posted similar content recently
	minHoursBetweenSimilar := s.configService.GetInt(ctx, "antispam.min_time_between_similar_hours", 24)
	cutoffTime := time.Now().Add(-time.Duration(minHoursBetweenSimilar) * time.Hour)

	for _, hash := range similarContent {
		if hash.UserID == userID && hash.CreatedAt.After(cutoffTime) {
			result.Allowed = false
			result.SetReason(fmt.Sprintf("Similar content posted recently (wait %d hours)", minHoursBetweenSimilar))

			retryAfter := time.Until(hash.CreatedAt.Add(time.Duration(minHoursBetweenSimilar) * time.Hour))
			result.SetRetryAfter(retryAfter)
			return nil
		}
	}

	return nil
}

// checkRapidActions checks for rapid successive actions and applies penalties if needed
func (s *Service) checkRapidActions(ctx context.Context, userID uuid.UUID, actionType models.ActivityType, result *models.AntiSpamCheck) error {
	rapidThreshold := s.configService.GetInt(ctx, "antispam.rapid_action_threshold", 5)
	windowMinutes := s.configService.GetInt(ctx, "antispam.rapid_action_window_minutes", 5)

	windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	count, err := s.repo.CountUserActivitiesSince(ctx, userID, actionType, windowStart)
	if err != nil {
		return err
	}

	if count >= rapidThreshold {
		// Apply progressive penalty
		penaltyMultiplier := s.configService.GetFloat64(ctx, "antispam.penalty_multiplier", 2.0)
		penaltyDurationHours := s.configService.GetInt(ctx, "antispam.penalty_duration_hours", 24)

		penalty := models.NewUserPenalty(
			userID,
			models.PenaltyTypeCooldown,
			fmt.Sprintf("Rapid %s actions detected (%d in %d minutes)", actionType, count, windowMinutes),
			penaltyMultiplier,
			time.Duration(penaltyDurationHours)*time.Hour,
			nil, // Automatic penalty
		)

		if err := s.repo.CreateUserPenalty(ctx, penalty); err != nil {
			return err
		}

		result.SetMetadata("penalty_applied", true)
		result.SetMetadata("penalty_reason", penalty.Reason)
		result.AddPenalty(penalty)

		// The penalty will be applied in the next check
	}

	return nil
}

// checkAnonymousLimits applies basic limits for anonymous users
func (s *Service) checkAnonymousLimits(ctx context.Context, ipAddress net.IP, actionType models.ActivityType) (*models.AntiSpamCheck, error) {
	result := models.NewAntiSpamCheck(true)

	var maxPerHour int
	var windowHours int = 1

	switch actionType {
	case models.ActivityTypePitchCreate:
		maxPerHour = s.configService.GetInt(ctx, "antispam.max_pitches_per_ip_per_hour", 20)
	case models.ActivityTypeVote:
		maxPerHour = 100 // More lenient for votes
	default:
		return result, nil // No limits for other actions
	}

	windowStart := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	count, err := s.repo.CountIPActivitiesSince(ctx, ipAddress, actionType, windowStart)
	if err != nil {
		return nil, err
	}

	if count >= maxPerHour {
		result.Allowed = false
		result.SetReason(fmt.Sprintf("IP rate limit exceeded (%d/%d per hour)", count, maxPerHour))

		// Calculate retry after (next hour)
		nextHour := time.Now().Truncate(time.Hour).Add(time.Hour)
		retryAfter := time.Until(nextHour)
		result.SetRetryAfter(retryAfter)
	}

	return result, nil
}

// generateContentHash creates a SHA256 hash of normalized content
func (s *Service) generateContentHash(content string) string {
	// Normalize content: lowercase, remove extra whitespace, remove punctuation
	normalized := strings.ToLower(content)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	normalized = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)

	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// CleanupExpiredPenalties removes expired penalties
func (s *Service) CleanupExpiredPenalties(ctx context.Context) error {
	return s.repo.CleanupExpiredPenalties(ctx)
}

// CleanupOldActivities removes old activity records (older than 30 days)
func (s *Service) CleanupOldActivities(ctx context.Context) error {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	return s.repo.CleanupOldActivities(ctx, cutoff)
}
