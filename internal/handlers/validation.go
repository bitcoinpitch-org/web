package handlers

import (
	"fmt"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/models"
)

// PitchInput represents the input data for creating or updating a pitch
type PitchInput struct {
	Content        string                `form:"content" json:"content"`
	Language       string                `form:"language" json:"language"`
	MainCategory   models.MainCategory   `form:"main_category" json:"main_category"`
	LengthCategory models.LengthCategory `form:"length_category" json:"length_category"`
	AuthorType     models.AuthorType     `form:"author_type" json:"author_type"`
	AuthorName     *string               `form:"author_name" json:"author_name,omitempty"`
	AuthorHandle   *string               `form:"author_handle" json:"author_handle,omitempty"`
	Tags           []string              `form:"tags" json:"tags,omitempty"`
}

// CalculateLengthCategory determines the appropriate length category based on content length and configurable limits
func CalculateLengthCategory(content string, configService *config.Service) models.LengthCategory {
	length := len(content)
	limits := configService.PitchLimits(nil) // context is optional

	switch {
	case length >= limits.OneLinerMin && length <= limits.OneLinerMax:
		return models.LengthCategoryOneLiner
	case length <= limits.SMSMax:
		return models.LengthCategorySMS
	case length <= limits.TweetMax:
		return models.LengthCategoryTweet
	case length <= limits.ElevatorMax:
		return models.LengthCategoryElevator
	default:
		return "" // Invalid/too long
	}
}

// ValidatePitchInput validates the pitch input data using configurable limits
func ValidatePitchInput(input PitchInput, configService *config.Service) error {
	// Validate content
	if input.Content == "" {
		return fmt.Errorf("content is required")
	}

	// Get current pitch limits from configuration
	limits := configService.PitchLimits(nil) // context is optional in this case

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

	// Validate language
	if input.Language == "" {
		return fmt.Errorf("language is required")
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
	case models.AuthorTypeSame, models.AuthorTypeUnknown:
		// No additional fields required
	case models.AuthorTypeCustom:
		if input.AuthorName == nil || *input.AuthorName == "" {
			return fmt.Errorf("author name required for custom author type")
		}
	case models.AuthorTypeTwitter:
		if input.AuthorHandle == nil || *input.AuthorHandle == "" {
			return fmt.Errorf("author handle required for Twitter author type")
		}
		// Validate Twitter handle format
		if !isValidTwitterHandle(*input.AuthorHandle) {
			return fmt.Errorf("invalid Twitter handle format")
		}
	case models.AuthorTypeNostr:
		if input.AuthorHandle == nil || *input.AuthorHandle == "" {
			return fmt.Errorf("author handle required for Nostr author type")
		}
		// Validate Nostr pubkey format
		if !isValidNostrPubkey(*input.AuthorHandle) {
			return fmt.Errorf("invalid Nostr pubkey format")
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
