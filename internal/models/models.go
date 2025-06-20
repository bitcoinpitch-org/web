package models

import (
	"time"

	"github.com/google/uuid"
)

// BaseModel contains common fields for all models
type BaseModel struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AuthType represents the authentication method used by a user
type AuthType string

const (
	AuthTypeTrezor   AuthType = "trezor"
	AuthTypeNostr    AuthType = "nostr"
	AuthTypeTwitter  AuthType = "twitter"
	AuthTypePassword AuthType = "password"
	AuthTypeEmail    AuthType = "email" // New email/password authentication
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleModerator UserRole = "moderator"
	UserRoleAdmin     UserRole = "admin"
)

// MainCategory represents the main category of a pitch
type MainCategory string

const (
	MainCategoryBitcoin   MainCategory = "bitcoin"
	MainCategoryLightning MainCategory = "lightning"
	MainCategoryCashu     MainCategory = "cashu"
)

// LengthCategory represents the length category of a pitch
type LengthCategory string

const (
	LengthCategoryOneLiner LengthCategory = "one-liner"
	LengthCategorySMS      LengthCategory = "sms"
	LengthCategoryTweet    LengthCategory = "tweet"
	LengthCategoryElevator LengthCategory = "elevator"
)

// AuthorType represents the type of author attribution for a pitch
type AuthorType string

const (
	AuthorTypeSame    AuthorType = "same"
	AuthorTypeUnknown AuthorType = "unknown"
	AuthorTypeCustom  AuthorType = "custom"
	AuthorTypeTwitter AuthorType = "twitter"
	AuthorTypeNostr   AuthorType = "nostr"
)

// VoteType represents the type of vote
type VoteType string

const (
	VoteTypeUp   VoteType = "up"
	VoteTypeDown VoteType = "down"
)
