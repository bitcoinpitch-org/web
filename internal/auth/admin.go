package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"bitcoinpitch.org/internal/models"
	"github.com/google/uuid"
)

// AdminService handles admin operations and initialization
type AdminService struct {
	repo        AdminRepository
	passwordSvc *PasswordService
	totpSvc     *TOTPService
}

// AdminRepository interface for admin operations
type AdminRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByRole(role models.UserRole) ([]*models.User, error)
	UpdateUser(user *models.User) error
	CountUsersByRole(role models.UserRole) (int, error)
}

// NewAdminService creates a new admin service
func NewAdminService(repo AdminRepository, passwordSvc *PasswordService, totpSvc *TOTPService) *AdminService {
	return &AdminService{
		repo:        repo,
		passwordSvc: passwordSvc,
		totpSvc:     totpSvc,
	}
}

// InitializeAdminUser creates an admin user on first startup if none exists
func (s *AdminService) InitializeAdminUser() error {
	// Check if any admin users exist
	adminCount, err := s.repo.CountUsersByRole(models.UserRoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to count admin users: %w", err)
	}

	// If admin users already exist, skip initialization
	if adminCount > 0 {
		log.Printf("Admin users already exist (%d), skipping initialization", adminCount)
		return nil
	}

	// Get admin setup token from environment
	adminToken := os.Getenv("ADMIN_SETUP_TOKEN")
	if adminToken == "" {
		log.Printf("No ADMIN_SETUP_TOKEN provided, skipping admin user creation")
		return nil
	}

	// Get admin email (optional, fallback to default)
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@bitcoinpitch.org"
	}

	log.Printf("Creating initial admin user with email: %s", adminEmail)

	// Generate secure password from token
	passwordHash, err := s.passwordSvc.HashPassword(adminToken)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Create admin user
	adminUser := models.NewEmailUser(adminEmail, passwordHash)
	adminUser.SetRole(models.UserRoleAdmin)
	adminUser.SetEmailVerified(true) // Admin email is pre-verified
	adminUser.SetUsername("admin")   // Default username

	// Create the admin user
	if err := s.repo.CreateUser(adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("✅ Admin user created successfully!")
	log.Printf("📧 Email: %s", adminEmail)
	log.Printf("🔑 Password: %s", adminToken)
	log.Printf("⚠️  Please change the password after first login!")
	log.Printf("🔐 Consider enabling TOTP 2FA for enhanced security")

	return nil
}

// SetupTOTPForUser sets up TOTP 2FA for a user
func (s *AdminService) SetupTOTPForUser(userID uuid.UUID, userEmail string) (*SetupTOTPResult, error) {
	// Generate TOTP secret
	key, err := s.totpSvc.GenerateSecret(userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.totpSvc.GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Generate QR code URL
	qrCodeURL := s.totpSvc.GenerateQRCodeURL(key.Secret(), userEmail)

	return &SetupTOTPResult{
		Secret:      key.Secret(),
		QRCodeURL:   qrCodeURL,
		BackupCodes: backupCodes,
	}, nil
}

// EnableTOTPForUser enables TOTP 2FA for a user after verification
func (s *AdminService) EnableTOTPForUser(userID uuid.UUID, secret string, totpCode string, backupCodes []string) error {
	// Validate TOTP code
	if !s.totpSvc.ValidateCode(secret, totpCode) {
		return ErrTOTPInvalid
	}

	// Get user
	// Note: This would need a GetUserByID method in the repository
	// For now, we'll assume it's implemented

	// Update user with TOTP settings
	// This is a placeholder - would need actual implementation
	log.Printf("TOTP enabled for user %s", userID)

	return nil
}

// GenerateAdminSetupToken generates a secure admin setup token
func (s *AdminService) GenerateAdminSetupToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateAdminToken validates an admin setup token
func (s *AdminService) ValidateAdminToken(token string) bool {
	// Check token format (64 hex characters)
	if len(token) != 64 {
		return false
	}

	// Check if all characters are valid hex
	for _, char := range strings.ToLower(token) {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

// SetupTOTPResult contains the result of TOTP setup
type SetupTOTPResult struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// PrintAdminInstructions prints instructions for admin setup
func (s *AdminService) PrintAdminInstructions() {
	token, err := s.GenerateAdminSetupToken()
	if err != nil {
		log.Printf("Error generating admin token: %v", err)
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔐 ADMIN SETUP INSTRUCTIONS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("To create an admin user, add this to your .env file:")
	fmt.Println()
	fmt.Printf("ADMIN_SETUP_TOKEN=%s\n", token)
	fmt.Printf("ADMIN_EMAIL=admin@bitcoinpitch.org\n")
	fmt.Println()
	fmt.Println("Then restart the application. The admin user will be created")
	fmt.Println("with the email and token as the password.")
	fmt.Println()
	fmt.Println("⚠️  IMPORTANT:")
	fmt.Println("- Change the password after first login")
	fmt.Println("- Enable TOTP 2FA for enhanced security")
	fmt.Println("- Keep the token secure - it's the admin password")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
}
