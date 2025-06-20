package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPService handles TOTP operations
type TOTPService struct {
	issuer string
}

// NewTOTPService creates a new TOTP service
func NewTOTPService(issuer string) *TOTPService {
	return &TOTPService{
		issuer: issuer,
	}
}

// GenerateSecret generates a new TOTP secret for a user
func (t *TOTPService) GenerateSecret(userEmail string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      t.issuer,
		AccountName: userEmail,
	})
}

// ValidateCode validates a TOTP code against a secret
func (t *TOTPService) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateQRCodeURL generates a QR code URL for the secret
func (t *TOTPService) GenerateQRCodeURL(secret, userEmail string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		url.QueryEscape(t.issuer),
		url.QueryEscape(userEmail),
		secret,
		url.QueryEscape(t.issuer),
	)
}

// GenerateBackupCodes generates backup codes for TOTP
func (t *TOTPService) GenerateBackupCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10 // Default 10 backup codes
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := t.generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code
	}

	return codes, nil
}

// generateBackupCode generates a single backup code
func (t *TOTPService) generateBackupCode() (string, error) {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode to base32 and take first 8 characters
	code := base32.StdEncoding.EncodeToString(bytes)
	code = strings.ToUpper(code[:8])

	// Format as XXXX-XXXX
	return fmt.Sprintf("%s-%s", code[:4], code[4:8]), nil
}

// ValidateBackupCode checks if a backup code is valid and removes it from the list
func (t *TOTPService) ValidateBackupCode(codes []string, inputCode string) (bool, []string) {
	// Normalize input code (remove spaces, dashes, make uppercase)
	inputCode = strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(inputCode, " ", ""), "-", ""))

	for i, code := range codes {
		// Normalize stored code
		normalizedCode := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(code, " ", ""), "-", ""))

		if normalizedCode == inputCode {
			// Remove the used backup code
			newCodes := make([]string, len(codes)-1)
			copy(newCodes[:i], codes[:i])
			copy(newCodes[i:], codes[i+1:])
			return true, newCodes
		}
	}

	return false, codes
}

// GetCurrentCode gets the current TOTP code for a secret (useful for testing)
func (t *TOTPService) GetCurrentCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
