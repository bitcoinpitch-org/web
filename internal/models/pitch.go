package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Pitch represents a pitch in the system
type Pitch struct {
	BaseModel
	UserID                  uuid.UUID      `json:"user_id" db:"user_id"`
	Content                 string         `json:"content" db:"content"`
	Language                string         `json:"language" db:"language"`
	MainCategory            MainCategory   `json:"main_category" db:"main_category"`
	LengthCategory          LengthCategory `json:"length_category" db:"length_category"`
	DeletedAt               *time.Time     `json:"deleted_at,omitempty" db:"deleted_at"`
	VoteCount               int            `json:"vote_count" db:"vote_count"`
	UpvoteCount             int            `json:"upvote_count" db:"upvote_count"`
	DownvoteCount           int            `json:"downvote_count" db:"downvote_count"`
	Score                   int            `json:"score" db:"score"`
	LastVoteAt              *time.Time     `json:"last_vote_at,omitempty" db:"last_vote_at"`
	LastEditAt              *time.Time     `json:"last_edit_at,omitempty" db:"last_edit_at"`
	PostedBy                uuid.UUID      `json:"posted_by" db:"posted_by"`
	AuthorType              AuthorType     `json:"author_type" db:"author_type"`
	AuthorName              *string        `json:"author_name,omitempty" db:"author_name"`
	AuthorHandle            *string        `json:"author_handle,omitempty" db:"author_handle"`
	Tags                    Tags           `json:"tags,omitempty" db:"tags"`
	PostedByDisplayName     *string        `json:"posted_by_display_name,omitempty" db:"posted_by_display_name"`
	PostedByAuthType        *AuthType      `json:"posted_by_auth_type,omitempty" db:"posted_by_auth_type"`
	PostedByUsername        *string        `json:"posted_by_username,omitempty" db:"posted_by_username"`
	PostedByShowAuthMethod  *bool          `json:"posted_by_show_auth_method,omitempty" db:"posted_by_show_auth_method"`
	PostedByShowUsername    *bool          `json:"posted_by_show_username,omitempty" db:"posted_by_show_username"`
	PostedByShowProfileInfo *bool          `json:"posted_by_show_profile_info,omitempty" db:"posted_by_show_profile_info"`
	// CurrentUser is set at runtime for template access, not stored in database
	CurrentUser *User `json:"-" db:"-"`
	// CurrentUserVote is set at runtime for template access, not stored in database
	CurrentUserVote *Vote `json:"-" db:"-"`
}

// NewPitch creates a new pitch with the given details
func NewPitch(userID, postedBy uuid.UUID, content, language string, mainCategory MainCategory, lengthCategory LengthCategory, authorType AuthorType) *Pitch {
	now := time.Now()
	return &Pitch{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:         userID,
		Content:        content,
		Language:       language,
		MainCategory:   mainCategory,
		LengthCategory: lengthCategory,
		PostedBy:       postedBy,
		AuthorType:     authorType,
	}
}

// SetAuthor sets the author details for the pitch
func (p *Pitch) SetAuthor(authorType AuthorType, authorName, authorHandle *string) {
	p.AuthorType = authorType
	p.AuthorName = authorName
	p.AuthorHandle = authorHandle
	p.UpdatedAt = time.Now()
}

// Edit updates the pitch content and resets votes
func (p *Pitch) Edit(content string) {
	p.Content = content
	now := time.Now()
	p.UpdatedAt = now
	p.LastEditAt = &now
	p.VoteCount = 0
	p.UpvoteCount = 0
	p.DownvoteCount = 0
	p.Score = 0
	p.LastVoteAt = nil
}

// Delete marks the pitch as deleted
func (p *Pitch) Delete() {
	now := time.Now()
	p.DeletedAt = &now
	p.UpdatedAt = now
}

// IsDeleted checks if the pitch is deleted
func (p *Pitch) IsDeleted() bool {
	return p.DeletedAt != nil
}

// GetPostedByDisplayName returns the display name respecting privacy settings
func (p *Pitch) GetPostedByDisplayName() string {
	// If user has disabled showing username, show Anonymous
	if p.PostedByShowUsername != nil && !*p.PostedByShowUsername {
		return "Anonymous"
	}

	// Show display name if available
	if p.PostedByDisplayName != nil && *p.PostedByDisplayName != "" {
		return *p.PostedByDisplayName
	}

	// Show username if available
	if p.PostedByUsername != nil && *p.PostedByUsername != "" {
		return *p.PostedByUsername
	}

	// Default to Anonymous
	return "Anonymous"
}

// GetPostedByPublicAuthType returns the auth type only if user allows it
func (p *Pitch) GetPostedByPublicAuthType() string {
	if p.PostedByShowAuthMethod != nil && *p.PostedByShowAuthMethod && p.PostedByAuthType != nil {
		switch *p.PostedByAuthType {
		case AuthTypeTrezor:
			return "Trezor"
		case AuthTypeNostr:
			return "Nostr"
		case AuthTypeTwitter:
			return "Twitter"
		case AuthTypePassword:
			return "Password"
		default:
			return ""
		}
	}
	return ""
}

// ShouldShowPostedByAuthMethod returns whether to show auth method
func (p *Pitch) ShouldShowPostedByAuthMethod() bool {
	return p.PostedByShowAuthMethod != nil && *p.PostedByShowAuthMethod
}

// ShouldShowPostedByUsername returns whether to show username
func (p *Pitch) ShouldShowPostedByUsername() bool {
	return p.PostedByShowUsername == nil || *p.PostedByShowUsername
}

// GetAuthorHandle returns the author handle as a string, safe for templates
func (p *Pitch) GetAuthorHandle() string {
	if p.AuthorHandle == nil {
		return ""
	}
	return *p.AuthorHandle
}

// GetAuthorHandleForTwitter returns the author handle without @ prefix for Twitter URLs
func (p *Pitch) GetAuthorHandleForTwitter() string {
	handle := p.GetAuthorHandle()
	if handle == "" {
		return ""
	}
	if strings.HasPrefix(handle, "@") {
		return handle[1:]
	}
	return handle
}

// Vote represents a vote on a pitch
type Vote struct {
	BaseModel
	PitchID  uuid.UUID `json:"pitch_id" db:"pitch_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	VoteType VoteType  `json:"vote_type" db:"vote_type"`
}

// NewVote creates a new vote
func NewVote(pitchID, userID uuid.UUID, voteType VoteType) *Vote {
	now := time.Now()
	return &Vote{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		PitchID:  pitchID,
		UserID:   userID,
		VoteType: voteType,
	}
}

// Tag represents a tag in the system
type Tag struct {
	BaseModel
	Name       string `json:"name" db:"name"`
	UsageCount int    `json:"usage_count" db:"usage_count"`
}

// NewTag creates a new tag
func NewTag(name string) *Tag {
	now := time.Now()
	return &Tag{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:       name,
		UsageCount: 0,
	}
}

// IncrementUsage increments the usage count of the tag
func (t *Tag) IncrementUsage() {
	t.UsageCount++
	t.UpdatedAt = time.Now()
}

// DecrementUsage decrements the usage count of the tag
func (t *Tag) DecrementUsage() {
	if t.UsageCount > 0 {
		t.UsageCount--
		t.UpdatedAt = time.Now()
	}
}

// PitchTag represents the many-to-many relationship between pitches and tags
type PitchTag struct {
	PitchID uuid.UUID `json:"pitch_id" db:"pitch_id"`
	TagID   uuid.UUID `json:"tag_id" db:"tag_id"`
	BaseModel
}

// Tags is a slice of Tag with a custom Scanner for JSON
// Tags implements sql.Scanner so sqlx can unmarshal the tags JSON column
// into the Tags field of Pitch

type Tags []Tag

func (t *Tags) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, t)
}
