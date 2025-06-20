package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents a user in the system
type User struct {
	BaseModel
	AuthType        AuthType `json:"auth_type" db:"auth_type"`
	AuthID          string   `json:"auth_id" db:"auth_id"`
	Username        *string  `json:"username,omitempty" db:"username"`
	DisplayName     *string  `json:"display_name,omitempty" db:"display_name"`
	ShowAuthMethod  bool     `json:"show_auth_method" db:"show_auth_method"`
	ShowUsername    bool     `json:"show_username" db:"show_username"`
	ShowProfileInfo bool     `json:"show_profile_info" db:"show_profile_info"`
	// Email registration fields
	Email                      *string    `json:"email,omitempty" db:"email"`
	PasswordHash               *string    `json:"-" db:"password_hash"` // Never expose in JSON
	EmailVerified              bool       `json:"email_verified" db:"email_verified"`
	EmailVerificationToken     *string    `json:"-" db:"email_verification_token"`
	EmailVerificationExpiresAt *time.Time `json:"-" db:"email_verification_expires_at"`
	// Role and permissions
	Role UserRole `json:"role" db:"role"`
	// TOTP 2FA fields
	TOTPSecret      *string        `json:"-" db:"totp_secret"`
	TOTPEnabled     bool           `json:"totp_enabled" db:"totp_enabled"`
	TOTPBackupCodes pq.StringArray `json:"-" db:"totp_backup_codes"`
	// Password reset fields
	PasswordResetToken     *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpiresAt *time.Time `json:"-" db:"password_reset_expires_at"`
	// Pagination preference
	PageSize *int `json:"page_size,omitempty" db:"page_size"`
}

// NewUser creates a new user with the given authentication details
func NewUser(authType AuthType, authID string) *User {
	now := time.Now()
	return &User{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		AuthType:        authType,
		AuthID:          authID,
		ShowAuthMethod:  false,        // Private by default
		ShowUsername:    true,         // Public by default
		ShowProfileInfo: false,        // Private by default
		Role:            UserRoleUser, // Default role
		EmailVerified:   false,        // Not verified by default
		TOTPEnabled:     false,        // TOTP disabled by default
	}
}

// NewEmailUser creates a new user with email/password authentication
func NewEmailUser(email, passwordHash string) *User {
	now := time.Now()
	return &User{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		AuthType:        AuthTypeEmail,
		AuthID:          email, // Use email as auth ID for email users
		Email:           &email,
		PasswordHash:    &passwordHash,
		ShowAuthMethod:  false,        // Private by default
		ShowUsername:    true,         // Public by default
		ShowProfileInfo: false,        // Private by default
		Role:            UserRoleUser, // Default role
		EmailVerified:   false,        // Not verified by default
		TOTPEnabled:     false,        // TOTP disabled by default
	}
}

// SetUsername sets the username for the user
func (u *User) SetUsername(username string) {
	u.Username = &username
	u.UpdatedAt = time.Now()
}

// SetDisplayName sets the display name for the user
func (u *User) SetDisplayName(displayName string) {
	u.DisplayName = &displayName
	u.UpdatedAt = time.Now()
}

// SetPrivacySettings updates privacy preferences
func (u *User) SetPrivacySettings(showAuthMethod, showUsername, showProfileInfo bool) {
	u.ShowAuthMethod = showAuthMethod
	u.ShowUsername = showUsername
	u.ShowProfileInfo = showProfileInfo
	u.UpdatedAt = time.Now()
}

// GetDisplayName returns the display name or username or a default value
func (u *User) GetDisplayName() string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	if u.Username != nil && *u.Username != "" {
		return *u.Username
	}
	return "Anonymous"
}

// GetPublicDisplayName returns the display name respecting privacy settings
func (u *User) GetPublicDisplayName() string {
	if !u.ShowUsername {
		return "Anonymous"
	}
	return u.GetDisplayName()
}

// GetAuthMethodString returns the authentication method as a string
func (u *User) GetAuthMethodString() string {
	switch u.AuthType {
	case AuthTypeTrezor:
		return "Trezor Hardware Wallet"
	case AuthTypeNostr:
		return "Nostr"
	case AuthTypeTwitter:
		return "Twitter / X"
	case AuthTypePassword:
		return "Username/Password"
	case AuthTypeEmail:
		return "Email/Password"
	default:
		return "Unknown"
	}
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsModerator returns true if the user has moderator role or higher
func (u *User) IsModerator() bool {
	return u.Role == UserRoleModerator || u.Role == UserRoleAdmin
}

// HasRole returns true if the user has the specified role or higher
func (u *User) HasRole(role UserRole) bool {
	switch role {
	case UserRoleUser:
		return true // All users have user role
	case UserRoleModerator:
		return u.Role == UserRoleModerator || u.Role == UserRoleAdmin
	case UserRoleAdmin:
		return u.Role == UserRoleAdmin
	default:
		return false
	}
}

// SetEmail sets the email for the user
func (u *User) SetEmail(email string) {
	u.Email = &email
	u.UpdatedAt = time.Now()
}

// SetPasswordHash sets the password hash for the user
func (u *User) SetPasswordHash(hash string) {
	u.PasswordHash = &hash
	u.UpdatedAt = time.Now()
}

// SetEmailVerified marks the email as verified
func (u *User) SetEmailVerified(verified bool) {
	u.EmailVerified = verified
	u.UpdatedAt = time.Now()
}

// SetEmailVerificationToken sets the email verification token
func (u *User) SetEmailVerificationToken(token string, expiresAt time.Time) {
	u.EmailVerificationToken = &token
	u.EmailVerificationExpiresAt = &expiresAt
	u.UpdatedAt = time.Now()
}

// ClearEmailVerificationToken clears the email verification token
func (u *User) ClearEmailVerificationToken() {
	u.EmailVerificationToken = nil
	u.EmailVerificationExpiresAt = nil
	u.UpdatedAt = time.Now()
}

// SetPasswordResetToken sets the password reset token
func (u *User) SetPasswordResetToken(token string, expiresAt time.Time) {
	u.PasswordResetToken = &token
	u.PasswordResetExpiresAt = &expiresAt
	u.UpdatedAt = time.Now()
}

// ClearPasswordResetToken clears the password reset token
func (u *User) ClearPasswordResetToken() {
	u.PasswordResetToken = nil
	u.PasswordResetExpiresAt = nil
	u.UpdatedAt = time.Now()
}

// SetTOTPSecret sets the TOTP secret for 2FA
func (u *User) SetTOTPSecret(secret string) {
	u.TOTPSecret = &secret
	u.UpdatedAt = time.Now()
}

// EnableTOTP enables TOTP 2FA for the user
func (u *User) EnableTOTP() {
	u.TOTPEnabled = true
	u.UpdatedAt = time.Now()
}

// DisableTOTP disables TOTP 2FA for the user
func (u *User) DisableTOTP() {
	u.TOTPEnabled = false
	u.TOTPSecret = nil
	u.TOTPBackupCodes = pq.StringArray{}
	u.UpdatedAt = time.Now()
}

// SetTOTPBackupCodes sets the TOTP backup codes
func (u *User) SetTOTPBackupCodes(codes []string) {
	u.TOTPBackupCodes = pq.StringArray(codes)
	u.UpdatedAt = time.Now()
}

// SetRole sets the user role
func (u *User) SetRole(role UserRole) {
	u.Role = role
	u.UpdatedAt = time.Now()
}

// SetPageSize sets the user's preferred page size
func (u *User) SetPageSize(pageSize int) {
	u.PageSize = &pageSize
	u.UpdatedAt = time.Now()
}

// GetPageSize returns the user's preferred page size or 0 if not set
func (u *User) GetPageSize() int {
	if u.PageSize != nil {
		return *u.PageSize
	}
	return 0
}

// ShouldShowAuthMethod returns whether to show auth method publicly
func (u *User) ShouldShowAuthMethod() bool {
	return u.ShowAuthMethod
}

// ShouldShowUsername returns whether to show username publicly
func (u *User) ShouldShowUsername() bool {
	return u.ShowUsername
}

// ShouldShowProfileInfo returns whether to show profile info publicly
func (u *User) ShouldShowProfileInfo() bool {
	return u.ShowProfileInfo
}

// Session represents a user session
type Session struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// NewSession creates a new session for a user
func NewSession(userID uuid.UUID, token string, expiresAt time.Time) *Session {
	now := time.Now()
	return &Session{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	Email     string    `json:"email" db:"email"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Used      bool      `json:"used" db:"used"`
}

// NewEmailVerificationToken creates a new email verification token
func NewEmailVerificationToken(userID uuid.UUID, token, email string, expiresAt time.Time) *EmailVerificationToken {
	now := time.Now()
	return &EmailVerificationToken{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:    userID,
		Token:     token,
		Email:     email,
		ExpiresAt: expiresAt,
		Used:      false,
	}
}

// IsExpired checks if the token has expired
func (t *EmailVerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// MarkAsUsed marks the token as used
func (t *EmailVerificationToken) MarkAsUsed() {
	t.Used = true
	t.UpdatedAt = time.Now()
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	Used      bool      `json:"used" db:"used"`
}

// NewPasswordResetToken creates a new password reset token
func NewPasswordResetToken(userID uuid.UUID, token string, expiresAt time.Time) *PasswordResetToken {
	now := time.Now()
	return &PasswordResetToken{
		BaseModel: BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		Used:      false,
	}
}

// IsExpired checks if the token has expired
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// MarkAsUsed marks the token as used
func (t *PasswordResetToken) MarkAsUsed() {
	t.Used = true
	t.UpdatedAt = time.Now()
}
