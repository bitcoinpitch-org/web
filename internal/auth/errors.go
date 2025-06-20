package auth

import "errors"

// Password validation errors
var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrPasswordTooLong  = errors.New("password must be no more than 128 characters long")
	ErrPasswordNoLetter = errors.New("password must contain at least one letter")
	ErrPasswordNoNumber = errors.New("password must contain at least one number")
)

// Email validation errors
var (
	ErrEmailInvalid     = errors.New("invalid email address")
	ErrEmailTaken       = errors.New("email address is already taken")
	ErrEmailNotVerified = errors.New("email address is not verified")
)

// Authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrTokenAlreadyUsed   = errors.New("token has already been used")
)

// TOTP errors
var (
	ErrTOTPInvalid    = errors.New("invalid TOTP code")
	ErrTOTPNotEnabled = errors.New("TOTP is not enabled for this user")
	ErrBackupCodeUsed = errors.New("backup code has already been used")
)

// Admin errors
var (
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrAdminRequired           = errors.New("admin privileges required")
	ErrModeratorRequired       = errors.New("moderator privileges required")
)
