package handlers

import (
	"bytes"
	"context"
	"image/png"
	"log"

	"bitcoinpitch.org/internal/auth"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/models"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TOTPHandler handles TOTP 2FA operations
type TOTPHandler struct {
	repo    *database.Repository
	totpSvc *auth.TOTPService
}

// NewTOTPHandler creates a new TOTP handler
func NewTOTPHandler(repo *database.Repository, totpSvc *auth.TOTPService) *TOTPHandler {
	return &TOTPHandler{
		repo:    repo,
		totpSvc: totpSvc,
	}
}

// GenerateTOTPSecret generates a new TOTP secret for the user
func (h *TOTPHandler) GenerateTOTPSecret(c *fiber.Ctx) error {
	log.Printf("[DEBUG] GenerateTOTPSecret: Starting request")

	// Get current user from session
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		log.Printf("[DEBUG] GenerateTOTPSecret: No user found in context")
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Authentication required",
		})
	}

	log.Printf("[DEBUG] GenerateTOTPSecret: User found: %s (auth type: %s)", user.GetDisplayName(), user.AuthType)

	// Check if user is using email or password authentication
	if user.AuthType != models.AuthTypeEmail && user.AuthType != models.AuthTypePassword {
		log.Printf("[DEBUG] GenerateTOTPSecret: Invalid auth type: %s", user.AuthType)
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "2FA is only available for email/password accounts",
		})
	}

	// Generate TOTP secret
	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	key, err := h.totpSvc.GenerateSecret(email)
	if err != nil {
		log.Printf("Error generating TOTP secret: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate TOTP secret",
		})
	}

	// Generate QR code URL
	qrURL := h.totpSvc.GenerateQRCodeURL(key.Secret(), email)

	log.Printf("[DEBUG] GenerateTOTPSecret: Success - generated secret for user %s", user.GetDisplayName())

	return c.JSON(fiber.Map{
		"success": true,
		"secret":  key.Secret(),
		"qr_url":  qrURL,
	})
}

// GenerateQRCode generates a QR code image for a given TOTP URL
func (h *TOTPHandler) GenerateQRCode(c *fiber.Ctx) error {
	// Get current user from session
	_, ok := c.Locals("user").(*models.User)
	if !ok {
		log.Printf("[DEBUG] GenerateQRCode: No user found in context")
		return c.Status(401).SendString("Authentication required")
	}

	// Get the TOTP URL from query parameter
	totpURL := c.Query("url")
	if totpURL == "" {
		log.Printf("[DEBUG] GenerateQRCode: No URL provided")
		return c.Status(400).SendString("TOTP URL is required")
	}

	log.Printf("[DEBUG] GenerateQRCode: Generating QR code for URL: %s", totpURL)

	// Create QR code
	qrCode, err := qr.Encode(totpURL, qr.M, qr.Auto)
	if err != nil {
		log.Printf("[ERROR] GenerateQRCode: Failed to encode QR: %v", err)
		return c.Status(500).SendString("Failed to generate QR code")
	}

	// Scale QR code to 200x200 pixels
	qrCode, err = barcode.Scale(qrCode, 200, 200)
	if err != nil {
		log.Printf("[ERROR] GenerateQRCode: Failed to scale QR: %v", err)
		return c.Status(500).SendString("Failed to scale QR code")
	}

	// Convert to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		log.Printf("[ERROR] GenerateQRCode: Failed to encode PNG: %v", err)
		return c.Status(500).SendString("Failed to encode QR code as PNG")
	}

	log.Printf("[DEBUG] GenerateQRCode: Successfully generated QR code (%d bytes)", buf.Len())

	// Set headers and return PNG image
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")

	return c.Send(buf.Bytes())
}

// EnableTOTP enables TOTP 2FA for the user after verification
func (h *TOTPHandler) EnableTOTP(c *fiber.Ctx) error {
	// Get current user from session
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(401).SendString("Authentication required")
	}

	// Parse form data
	secret := c.FormValue("totp_secret")
	totpCode := c.FormValue("totp_code")

	if secret == "" || totpCode == "" {
		return c.Status(400).SendString("TOTP secret and code are required")
	}

	// Validate TOTP code
	if !h.totpSvc.ValidateCode(secret, totpCode) {
		return c.Status(400).SendString("Invalid TOTP code")
	}

	// Check if user is using email or password authentication
	if user.AuthType != models.AuthTypeEmail && user.AuthType != models.AuthTypePassword {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "2FA is only available for email/password accounts",
		})
	}

	// Generate backup codes
	backupCodes, err := h.totpSvc.GenerateBackupCodes(10)
	if err != nil {
		log.Printf("Error generating backup codes: %v", err)
		return c.Status(500).SendString("Failed to generate backup codes")
	}

	// Enable TOTP for user
	user.SetTOTPSecret(secret)
	user.SetTOTPBackupCodes(backupCodes)
	user.EnableTOTP()

	// Update user in database
	if err := h.repo.UpdateUser(context.Background(), user); err != nil {
		log.Printf("Error updating user: %v", err)
		return c.Status(500).SendString("Failed to enable 2FA")
	}

	log.Printf("TOTP 2FA enabled for user %s", user.BaseModel.ID)

	// Redirect back to profile with success message
	return c.Redirect("/user/profile?totp_enabled=1")
}

// DisableTOTP disables TOTP 2FA for the user
func (h *TOTPHandler) DisableTOTP(c *fiber.Ctx) error {
	// Get current user from session
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(401).SendString("Authentication required")
	}

	// Parse form data
	totpCode := c.FormValue("totp_code")
	if totpCode == "" {
		return c.Status(400).SendString("TOTP code is required")
	}

	// Check if TOTP is enabled
	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return c.Status(400).SendString("2FA is not enabled")
	}

	// Validate TOTP code
	if !h.totpSvc.ValidateCode(*user.TOTPSecret, totpCode) {
		return c.Status(400).SendString("Invalid TOTP code")
	}

	// Disable TOTP for user
	user.DisableTOTP()

	// Update user in database
	if err := h.repo.UpdateUser(context.Background(), user); err != nil {
		log.Printf("Error updating user: %v", err)
		return c.Status(500).SendString("Failed to disable 2FA")
	}

	log.Printf("TOTP 2FA disabled for user %s", user.BaseModel.ID)

	// Redirect back to profile with success message
	return c.Redirect("/user/profile?totp_disabled=1")
}

// GetBackupCodes returns the user's backup codes
func (h *TOTPHandler) GetBackupCodes(c *fiber.Ctx) error {
	// Get current user from session
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(401).JSON(fiber.Map{
			"success": false,
			"error":   "Authentication required",
		})
	}

	// Check if TOTP is enabled
	if !user.TOTPEnabled {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "2FA is not enabled",
		})
	}

	// Return backup codes
	return c.JSON(fiber.Map{
		"success": true,
		"codes":   []string(user.TOTPBackupCodes),
	})
}

// ValidateTOTPLogin validates TOTP code during login
func (h *TOTPHandler) ValidateTOTPLogin(userID uuid.UUID, code string) error {
	// Get user from database
	user, err := h.repo.GetUserByID(context.Background(), userID)
	if err != nil {
		return err
	}

	// Check if TOTP is enabled
	if !user.TOTPEnabled || user.TOTPSecret == nil {
		return auth.ErrTOTPNotEnabled
	}

	// Check if it's a backup code
	if len(code) > 6 {
		// Try backup code
		valid, newCodes := h.totpSvc.ValidateBackupCode([]string(user.TOTPBackupCodes), code)
		if valid {
			// Update backup codes (remove used code)
			user.SetTOTPBackupCodes(newCodes)
			if err := h.repo.UpdateUser(context.Background(), user); err != nil {
				log.Printf("Error updating backup codes: %v", err)
			}
			return nil
		}
		return auth.ErrTOTPInvalid
	}

	// Validate TOTP code
	if !h.totpSvc.ValidateCode(*user.TOTPSecret, code) {
		return auth.ErrTOTPInvalid
	}

	return nil
}

// CheckTOTPRequired checks if user has 2FA enabled and returns requirements
func (h *TOTPHandler) CheckTOTPRequired(userID uuid.UUID) (bool, error) {
	user, err := h.repo.GetUserByID(context.Background(), userID)
	if err != nil {
		return false, err
	}

	return user.TOTPEnabled, nil
}
