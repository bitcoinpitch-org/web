package validation

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
		if !IsValidTwitterHandle(*input.AuthorHandle) {
			return fmt.Errorf("invalid Twitter handle format")
		}
	case models.AuthorTypeNostr:
		if input.AuthorHandle == nil || *input.AuthorHandle == "" {
			return fmt.Errorf("author handle required for Nostr author type")
		}
		// Validate Nostr pubkey format
		if !IsValidNostrPubkey(*input.AuthorHandle) {
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
		if !IsValidTag(tag) {
			return fmt.Errorf("invalid tag format: %s", tag)
		}
	}

	return nil
}

// IsValidTwitterHandle validates a Twitter handle
func IsValidTwitterHandle(handle string) bool {
	// Twitter handles are 1-15 characters, alphanumeric + underscore
	// Must start with @
	return len(handle) >= 2 && len(handle) <= 16 && handle[0] == '@' && IsValidTwitterUsername(handle[1:])
}

// IsValidTwitterUsername validates a Twitter username
func IsValidTwitterUsername(username string) bool {
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// IsValidNostrPubkey validates a Nostr pubkey
func IsValidNostrPubkey(pubkey string) bool {
	// Nostr pubkeys are npub1 followed by 58 base58 characters
	return len(pubkey) == 63 && pubkey[:5] == "npub1" && IsValidBase58(pubkey[5:])
}

// IsValidBase58 validates a base58 string
func IsValidBase58(s string) bool {
	for _, c := range s {
		if !((c >= '1' && c <= '9') || (c >= 'A' && c <= 'H') || (c >= 'J' && c <= 'N') || (c >= 'P' && c <= 'Z') || (c >= 'a' && c <= 'k') || (c >= 'm' && c <= 'z')) {
			return false
		}
	}
	return true
}

// IsValidTag validates a tag
func IsValidTag(tag string) bool {
	// Tags are 1-50 characters, alphanumeric + underscore + hyphen
	if len(tag) < 1 || len(tag) > 50 {
		return false
	}
	for _, c := range tag {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
