package middleware

import (
	"fmt"
	"log"
	"net"
	"time"

	"bitcoinpitch.org/internal/antispam"
	"bitcoinpitch.org/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AntiSpamMiddleware creates middleware for antispam protection
func AntiSpamMiddleware(antispamSvc *antispam.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Set antispam service in context for handlers
		c.Locals("antispamService", antispamSvc)
		return c.Next()
	}
}

// CheckPitchCreationLimit checks if user can create a pitch
func CheckPitchCreationLimit(c *fiber.Ctx, content string) error {
	antispamSvc := c.Locals("antispamService").(*antispam.Service)

	// Get user (may be nil for anonymous)
	var userID *uuid.UUID
	if user, ok := c.Locals("user").(*models.User); ok {
		userID = &user.BaseModel.ID
	}

	// Get IP and User-Agent
	ipAddress := net.ParseIP(c.IP())
	userAgent := c.Get("User-Agent")

	// Check antispam rules
	check, err := antispamSvc.CheckPitchCreation(c.Context(), userID, content, ipAddress, userAgent)
	if err != nil {
		log.Printf("Antispam check error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to verify request",
		})
	}

	if !check.Allowed {
		status := fiber.StatusTooManyRequests
		response := fiber.Map{
			"error":   check.Reason,
			"blocked": true,
		}

		if check.RetryAfter != nil {
			response["retry_after_seconds"] = int(check.RetryAfter.Seconds())
			response["retry_after_human"] = formatDuration(*check.RetryAfter)
		}

		if len(check.Penalties) > 0 {
			response["penalties"] = check.Penalties
		}

		return c.Status(status).JSON(response)
	}

	// Record the activity for tracking
	metadata := map[string]interface{}{
		"content_length": len(content),
		"ip_address":     c.IP(),
	}

	go func() {
		// Record activity asynchronously to not block the request
		err := antispamSvc.RecordActivity(
			c.Context(),
			userID,
			models.ActivityTypePitchCreate,
			nil, // No target ID for creation
			ipAddress,
			userAgent,
			metadata,
		)
		if err != nil {
			log.Printf("Failed to record activity: %v", err)
		}
	}()

	return nil
}

// CheckPitchEditLimit checks if user can edit a pitch
func CheckPitchEditLimit(c *fiber.Ctx, userID uuid.UUID, pitchID uuid.UUID, content string) error {
	antispamSvc := c.Locals("antispamService").(*antispam.Service)

	// Get IP and User-Agent
	ipAddress := net.ParseIP(c.IP())
	userAgent := c.Get("User-Agent")

	// Check antispam rules
	check, err := antispamSvc.CheckPitchEdit(c.Context(), userID, pitchID, content, ipAddress, userAgent)
	if err != nil {
		log.Printf("Antispam check error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to verify request",
		})
	}

	if !check.Allowed {
		status := fiber.StatusTooManyRequests
		response := fiber.Map{
			"error":   check.Reason,
			"blocked": true,
		}

		if check.RetryAfter != nil {
			response["retry_after_seconds"] = int(check.RetryAfter.Seconds())
			response["retry_after_human"] = formatDuration(*check.RetryAfter)
		}

		if len(check.Penalties) > 0 {
			response["penalties"] = check.Penalties
		}

		return c.Status(status).JSON(response)
	}

	// Record the activity for tracking
	metadata := map[string]interface{}{
		"content_length": len(content),
		"pitch_id":       pitchID.String(),
		"ip_address":     c.IP(),
	}

	go func() {
		// Record activity asynchronously
		err := antispamSvc.RecordActivity(
			c.Context(),
			&userID,
			models.ActivityTypePitchEdit,
			&pitchID,
			ipAddress,
			userAgent,
			metadata,
		)
		if err != nil {
			log.Printf("Failed to record activity: %v", err)
		}
	}()

	return nil
}

// CheckVoteLimit checks if user can vote
func CheckVoteLimit(c *fiber.Ctx, pitchID uuid.UUID) error {
	antispamSvc := c.Locals("antispamService").(*antispam.Service)

	// Get user (may be nil for anonymous)
	var userID *uuid.UUID
	if user, ok := c.Locals("user").(*models.User); ok {
		userID = &user.BaseModel.ID
	}

	// Get IP and User-Agent
	ipAddress := net.ParseIP(c.IP())
	userAgent := c.Get("User-Agent")

	// Check antispam rules
	check, err := antispamSvc.CheckVote(c.Context(), userID, pitchID, ipAddress, userAgent)
	if err != nil {
		log.Printf("Antispam check error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to verify request",
		})
	}

	if !check.Allowed {
		status := fiber.StatusTooManyRequests
		response := fiber.Map{
			"error":   check.Reason,
			"blocked": true,
		}

		if check.RetryAfter != nil {
			response["retry_after_seconds"] = int(check.RetryAfter.Seconds())
			response["retry_after_human"] = formatDuration(*check.RetryAfter)
		}

		return c.Status(status).JSON(response)
	}

	// Record the activity for tracking
	metadata := map[string]interface{}{
		"pitch_id":   pitchID.String(),
		"ip_address": c.IP(),
	}

	go func() {
		// Record activity asynchronously
		err := antispamSvc.RecordActivity(
			c.Context(),
			userID,
			models.ActivityTypeVote,
			&pitchID,
			ipAddress,
			userAgent,
			metadata,
		)
		if err != nil {
			log.Printf("Failed to record activity: %v", err)
		}
	}()

	return nil
}

// RecordContentHash records content hash for duplicate detection
func RecordContentHash(c *fiber.Ctx, userID uuid.UUID, content string, pitchID uuid.UUID) {
	antispamSvc := c.Locals("antispamService").(*antispam.Service)

	go func() {
		// Record content hash asynchronously
		err := antispamSvc.RecordContentHash(c.Context(), userID, content, pitchID)
		if err != nil {
			log.Printf("Failed to record content hash: %v", err)
		}
	}()
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
