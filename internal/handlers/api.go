package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// APIPitchesListHandler returns a list of pitches
func APIPitchesListHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Parse query parameters
	limit := 10 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filters from query parameters
	filters := make(map[string]interface{})
	if category := c.Query("category"); category != "" {
		filters["main_category"] = category
	}
	if language := c.Query("language"); language != "" {
		filters["language"] = language
	}

	// Get pitches from database
	pitches, err := repo.ListPitches(c.Context(), filters, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pitches: " + err.Error(),
		})
	}

	// Return JSON response
	return c.JSON(fiber.Map{
		"pitches": pitches,
		"meta": fiber.Map{
			"limit":  limit,
			"offset": offset,
			"total":  len(pitches), // TODO: Add total count query
		},
	})
}

// APIPitchGetHandler returns a single pitch
func APIPitchGetHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pitch ID",
		})
	}

	// Get pitch from database
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		if err == database.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pitch not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pitch: " + err.Error(),
		})
	}

	// Return JSON response
	return c.JSON(pitch)
}

// APIPitchCreateHandler handles the POST request for creating a new pitch via the API
func APIPitchCreateHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get config service from context
	configService := c.Locals("configService").(*config.Service)

	// Get user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Parse request body
	var input PitchInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request data: " + err.Error(),
		})
	}

	// Calculate length category based on content length using configurable limits
	input.LengthCategory = CalculateLengthCategory(input.Content, configService)

	// Validate input using configurable limits
	if err := ValidatePitchInput(input, configService); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create pitch
	pitch := models.NewPitch(
		userID,
		userID, // posted_by is same as user_id for now
		input.Content,
		input.Language,
		input.MainCategory,
		input.LengthCategory,
		input.AuthorType,
	)

	// Set author details if provided
	if input.AuthorName != nil || input.AuthorHandle != nil {
		pitch.SetAuthor(input.AuthorType, input.AuthorName, input.AuthorHandle)
	}

	// Add tags if provided
	if len(input.Tags) > 0 {
		pitch.Tags = make([]models.Tag, len(input.Tags))
		for i, tagName := range input.Tags {
			pitch.Tags[i] = *models.NewTag(tagName)
		}
	}

	// Save to database
	if err := repo.CreatePitch(c.Context(), pitch); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create pitch: " + err.Error(),
		})
	}

	// Return JSON response
	return c.Status(fiber.StatusCreated).JSON(pitch)
}

// APIPitchUpdateHandler updates an existing pitch
func APIPitchUpdateHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pitch ID",
		})
	}

	// Get existing pitch
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		if err == database.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pitch not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pitch: " + err.Error(),
		})
	}

	// Check ownership
	if pitch.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to edit this pitch",
		})
	}

	// Parse request body
	var input struct {
		Content        string                `json:"content"`
		Language       string                `json:"language"`
		MainCategory   models.MainCategory   `json:"main_category"`
		LengthCategory models.LengthCategory `json:"length_category"`
		AuthorType     models.AuthorType     `json:"author_type"`
		AuthorName     *string               `json:"author_name,omitempty"`
		AuthorHandle   *string               `json:"author_handle,omitempty"`
		Tags           []string              `json:"tags,omitempty"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request data: " + err.Error(),
		})
	}

	// Validate input
	if err := validatePitchInput(input, c.Locals("configService").(*config.Service)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Update pitch
	pitch.Content = input.Content
	pitch.Language = input.Language
	pitch.MainCategory = input.MainCategory
	pitch.LengthCategory = input.LengthCategory
	pitch.SetAuthor(input.AuthorType, input.AuthorName, input.AuthorHandle)

	// Reset votes on edit
	pitch.Edit(input.Content)

	// Update tags if provided
	if len(input.Tags) > 0 {
		pitch.Tags = make([]models.Tag, len(input.Tags))
		for i, tagName := range input.Tags {
			pitch.Tags[i] = *models.NewTag(tagName)
		}
	}

	// Save to database
	if err := repo.UpdatePitch(c.Context(), pitch); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update pitch: " + err.Error(),
		})
	}

	// Return JSON response
	return c.JSON(pitch)
}

// APIPitchDeleteHandler deletes a pitch
func APIPitchDeleteHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pitch ID",
		})
	}

	// Get existing pitch
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		if err == database.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Pitch not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pitch: " + err.Error(),
		})
	}

	// Check ownership
	if pitch.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to delete this pitch",
		})
	}

	// Delete pitch
	if err := repo.DeletePitch(c.Context(), pitchID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete pitch: " + err.Error(),
		})
	}

	// Return success response
	return c.JSON(fiber.Map{
		"message": "Pitch deleted successfully",
		"id":      pitchID,
	})
}

// APIPitchVoteHandler handles voting on pitches
func APIPitchVoteHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get user from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pitch ID",
		})
	}

	// Parse vote type
	voteType := models.VoteType(c.FormValue("type"))
	if voteType != models.VoteTypeUp && voteType != models.VoteTypeDown {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid vote type",
		})
	}

	// Get existing vote if any
	existingVote, err := repo.GetVote(c.Context(), pitchID, userID)
	if err != nil && err != database.ErrNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing vote: " + err.Error(),
		})
	}

	// Handle vote change or new vote
	if existingVote != nil {
		// If same vote type, remove vote
		if existingVote.VoteType == voteType {
			if err := repo.DeleteVote(c.Context(), existingVote); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to remove vote: " + err.Error(),
				})
			}
		} else {
			// If different vote type, update vote
			existingVote.VoteType = voteType
			existingVote.UpdatedAt = time.Now()
			if err := repo.UpdateVote(c.Context(), existingVote); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to update vote: " + err.Error(),
				})
			}
		}
	} else {
		// Create new vote
		vote := models.NewVote(pitchID, userID, voteType)
		if err := repo.CreateVote(c.Context(), vote); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create vote: " + err.Error(),
			})
		}
	}

	// Get updated pitch
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch updated pitch: " + err.Error(),
		})
	}

	// Return JSON response with updated vote counts
	return c.JSON(fiber.Map{
		"message": "Vote recorded successfully",
		"pitch": fiber.Map{
			"id":             pitch.ID,
			"vote_count":     pitch.VoteCount,
			"upvote_count":   pitch.UpvoteCount,
			"downvote_count": pitch.DownvoteCount,
			"score":          pitch.Score,
		},
	})
}

// Helper function to validate pitch input using configurable limits
func validatePitchInput(input struct {
	Content        string                `json:"content"`
	Language       string                `json:"language"`
	MainCategory   models.MainCategory   `json:"main_category"`
	LengthCategory models.LengthCategory `json:"length_category"`
	AuthorType     models.AuthorType     `json:"author_type"`
	AuthorName     *string               `json:"author_name,omitempty"`
	AuthorHandle   *string               `json:"author_handle,omitempty"`
	Tags           []string              `json:"tags,omitempty"`
}, configService *config.Service) error {
	// Get current pitch limits from configuration
	limits := configService.PitchLimits(nil) // context is optional

	// Validate content length based on category
	switch input.LengthCategory {
	case models.LengthCategoryOneLiner:
		if len(input.Content) < limits.OneLinerMin || len(input.Content) > limits.OneLinerMax {
			return fmt.Errorf("one-liner must be between %d and %d characters", limits.OneLinerMin, limits.OneLinerMax)
		}
	case models.LengthCategorySMS:
		if len(input.Content) > limits.SMSMax {
			return fmt.Errorf("SMS must be at most %d characters", limits.SMSMax)
		}
	case models.LengthCategoryTweet:
		if len(input.Content) > limits.TweetMax {
			return fmt.Errorf("tweet must be at most %d characters", limits.TweetMax)
		}
	case models.LengthCategoryElevator:
		if len(input.Content) > limits.ElevatorMax {
			return fmt.Errorf("elevator pitch must be at most %d characters", limits.ElevatorMax)
		}
	default:
		return fmt.Errorf("invalid length category")
	}

	// Validate main category
	switch input.MainCategory {
	case models.MainCategoryBitcoin, models.MainCategoryLightning, models.MainCategoryCashu:
		// Valid categories
	default:
		return fmt.Errorf("invalid main category")
	}

	// Validate author type and required fields
	switch input.AuthorType {
	case models.AuthorTypeSame:
		// No additional fields required
	case models.AuthorTypeUnknown:
		// No additional fields required
	case models.AuthorTypeCustom:
		if input.AuthorName == nil || *input.AuthorName == "" {
			return fmt.Errorf("author name required for custom author type")
		}
	case models.AuthorTypeTwitter:
		if input.AuthorHandle == nil || *input.AuthorHandle == "" {
			return fmt.Errorf("author handle required for Twitter author type")
		}
	case models.AuthorTypeNostr:
		if input.AuthorHandle == nil || *input.AuthorHandle == "" {
			return fmt.Errorf("author handle required for Nostr author type")
		}
	default:
		return fmt.Errorf("invalid author type")
	}

	// Validate tags
	if len(input.Tags) > 5 {
		return fmt.Errorf("maximum 5 tags allowed")
	}
	for _, tag := range input.Tags {
		if !isValidTag(tag) {
			return fmt.Errorf("invalid tag format: %s", tag)
		}
	}

	return nil
}

// APILanguageUsageHandler returns language usage statistics
func APILanguageUsageHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get language usage statistics
	usage, err := repo.GetLanguageUsage(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch language usage: " + err.Error(),
		})
	}

	// Return JSON response
	return c.JSON(usage)
}

// APISearchHandler performs full-text search on pitches
func APISearchHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get search query
	query := c.Query("q", "")
	if strings.TrimSpace(query) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Search query is required",
		})
	}

	// Parse pagination parameters
	limit := 25 // default limit for search
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build filters from query parameters
	filters := make(map[string]interface{})
	if category := c.Query("category"); category != "" {
		filters["main_category"] = category
	}
	if language := c.Query("language"); language != "" {
		filters["language"] = language
	}
	if lengthCategory := c.Query("length"); lengthCategory != "" {
		filters["length_category"] = lengthCategory
	}

	// Check if there's a tag filter as well
	tagFilter := c.Query("tag", "")

	var pitches []*models.Pitch
	var totalCount int
	var err error

	if tagFilter != "" && filters["main_category"] != nil {
		// Search with tag filter
		category := filters["main_category"].(string)
		delete(filters, "main_category") // Remove from filters as it's handled separately

		pitches, err = repo.SearchPitchesByTagAndFilters(c.Context(), query, category, tagFilter, filters, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search pitches: " + err.Error(),
			})
		}

		totalCount, err = repo.CountSearchPitchesByTagAndFilters(c.Context(), query, category, tagFilter, filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count search results: " + err.Error(),
			})
		}
	} else {
		// Regular search
		pitches, err = repo.SearchPitches(c.Context(), query, filters, limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search pitches: " + err.Error(),
			})
		}

		totalCount, err = repo.CountSearchPitches(c.Context(), query, filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to count search results: " + err.Error(),
			})
		}
	}

	// Return JSON response
	return c.JSON(fiber.Map{
		"pitches": pitches,
		"query":   query,
		"meta": fiber.Map{
			"limit":        limit,
			"offset":       offset,
			"total_count":  totalCount,
			"total_pages":  (totalCount + limit - 1) / limit,
			"current_page": (offset / limit) + 1,
		},
		"filters": filters,
	})
}
