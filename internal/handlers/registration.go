package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"bitcoinpitch.org/internal/auth"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/email"
	"bitcoinpitch.org/internal/models"
	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
)

// RegistrationHandler handles user registration
type RegistrationHandler struct {
	repo        UserRepository
	passwordSvc *auth.PasswordService
	emailSvc    *email.Service
	totpSvc     *auth.TOTPService
}

// UserRepository interface for user operations
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByEmailVerificationToken(token string) (*models.User, error)
	UpdateUser(user *models.User) error
	CreateEmailVerificationToken(token *models.EmailVerificationToken) error
	GetEmailVerificationToken(token string) (*models.EmailVerificationToken, error)
	UpdateEmailVerificationToken(token *models.EmailVerificationToken) error
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(
	repo UserRepository,
	passwordSvc *auth.PasswordService,
	emailSvc *email.Service,
	totpSvc *auth.TOTPService,
) *RegistrationHandler {
	return &RegistrationHandler{
		repo:        repo,
		passwordSvc: passwordSvc,
		emailSvc:    emailSvc,
		totpSvc:     totpSvc,
	}
}

// ShowRegistrationForm shows the registration form
func (h *RegistrationHandler) ShowRegistrationForm(c *fiber.Ctx) error {
	return c.Render("pages/register", fiber.Map{
		"Title":        "Register - BitcoinPitch.org",
		"ShowRegister": true,
	})
}

// RegisterUser handles user registration
func (h *RegistrationHandler) RegisterUser(c *fiber.Ctx) error {
	// Parse form data
	email := strings.TrimSpace(strings.ToLower(c.FormValue("email")))
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")
	username := strings.TrimSpace(c.FormValue("username"))

	// Validate input
	if email == "" {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Email is required",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	if password == "" {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Password is required",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	if password != confirmPassword {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Passwords do not match",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Validate email format
	if _, err := mail.ParseAddress(email); err != nil {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Invalid email address",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Validate password strength
	if err := h.passwordSvc.ValidatePasswordStrength(password); err != nil {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        err.Error(),
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Check if email already exists
	existingUser, err := h.repo.GetUserByEmail(email)
	if err == nil && existingUser != nil {
		return c.Status(400).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Email address is already taken",
			"Email":        "",
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Hash password
	passwordHash, err := h.passwordSvc.HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return c.Status(500).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Registration failed. Please try again.",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Create user
	user := models.NewEmailUser(email, passwordHash)
	if username != "" {
		user.SetUsername(username)
	}

	// Generate verification token
	token, err := h.generateSecureToken()
	if err != nil {
		log.Printf("Error generating verification token: %v", err)
		return c.Status(500).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Registration failed. Please try again.",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Set verification token (expires in 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)
	user.SetEmailVerificationToken(token, expiresAt)

	// Save user
	if err := h.repo.CreateUser(user); err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Status(500).Render("pages/register", fiber.Map{
			"Title":        "Register - BitcoinPitch.org",
			"Error":        "Registration failed. Please try again.",
			"Email":        email,
			"Username":     username,
			"ShowRegister": true,
		})
	}

	// Create verification token record
	verificationToken := models.NewEmailVerificationToken(
		user.ID,
		token,
		email,
		expiresAt,
	)

	if err := h.repo.CreateEmailVerificationToken(verificationToken); err != nil {
		log.Printf("Error creating verification token: %v", err)
		// Continue anyway, user can request new token later
	}

	// Send verification email
	siteURL := os.Getenv("SITE_URL")
	log.Printf("🔍 DEBUG: Raw SITE_URL from environment: '%s'", siteURL)
	if siteURL == "" {
		siteURL = "http://localhost:8090" // fallback for development
		log.Printf("🔍 DEBUG: Using fallback URL: '%s'", siteURL)
	} else {
		log.Printf("🔍 DEBUG: Using environment SITE_URL: '%s'", siteURL)
	}
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", siteURL, token)
	log.Printf("🔍 DEBUG: Generated verification URL: '%s'", verificationURL)

	displayName := username
	if displayName == "" {
		displayName = email
	}

	log.Printf("🔍 DEBUG: About to send verification email to: %s", email)
	if err := h.emailSvc.SendVerificationEmail(email, displayName, verificationURL); err != nil {
		log.Printf("Error sending verification email: %v", err)
		// Don't fail registration, just show a different message
		return c.Render("pages/register-success", fiber.Map{
			"Title":      "Registration Successful - BitcoinPitch.org",
			"Email":      email,
			"EmailError": "Registration successful, but we couldn't send the verification email. Please contact support.",
		})
	}
	log.Printf("🔍 DEBUG: Verification email sent successfully")

	return c.Render("pages/register-success", fiber.Map{
		"Title": "Registration Successful - BitcoinPitch.org",
		"Email": email,
	})
}

// VerifyEmail handles email verification
func (h *RegistrationHandler) VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(400).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Invalid verification link",
		})
	}

	// Get verification token
	verificationToken, err := h.repo.GetEmailVerificationToken(token)
	if err != nil {
		return c.Status(400).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Invalid verification link",
		})
	}

	// Check if token is expired
	if verificationToken.IsExpired() {
		return c.Status(400).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Verification link has expired. Please request a new one.",
		})
	}

	// Check if token is already used
	if verificationToken.Used {
		return c.Status(400).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Verification link has already been used.",
		})
	}

	// Get user by token
	user, err := h.repo.GetUserByEmailVerificationToken(token)
	if err != nil {
		return c.Status(400).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Invalid verification link",
		})
	}

	// Mark email as verified
	user.SetEmailVerified(true)
	user.ClearEmailVerificationToken()

	// Mark token as used
	verificationToken.MarkAsUsed()

	// Update both records
	if err := h.repo.UpdateUser(user); err != nil {
		log.Printf("Error updating user: %v", err)
		return c.Status(500).Render("pages/verify-email-error", fiber.Map{
			"Title": "Email Verification - BitcoinPitch.org",
			"Error": "Verification failed. Please try again.",
		})
	}

	if err := h.repo.UpdateEmailVerificationToken(verificationToken); err != nil {
		log.Printf("Error updating verification token: %v", err)
		// Continue anyway, email is verified
	}

	return c.Render("pages/verify-email-success", fiber.Map{
		"Title": "Email Verified - BitcoinPitch.org",
		"Email": user.Email,
	})
}

// generateSecureToken generates a secure random token
func (h *RegistrationHandler) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RegisterPageHandler shows the registration form
func RegisterPageHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)

	// Get template
	tmpl, err := view.GetTemplate("pages/register.jet")
	if err != nil {
		log.Printf("Error loading register template: %v", err)
		return c.Status(500).SendString("Internal Server Error")
	}

	// Create template variables
	vars := make(jet.VarMap)
	vars.Set("CsrfToken", c.Locals("csrf"))

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}
	vars.Set("ShowUserMenu", false) // Not authenticated on register page

	// Add i18n translation function
	if t, ok := c.Locals("t").(func(string, ...interface{}) string); ok {
		vars.Set("t", t)
	}

	// Render template
	return renderTemplate(c, tmpl, vars)
}

// VerifyEmailHandler handles email verification via token
func VerifyEmailHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	repo := c.Locals("repo").(*database.Repository)

	token := c.Query("token")
	if token == "" {
		return renderVerifyEmailPage(c, view, "verify.error_invalid_token", "")
	}

	// Verify email token
	verificationToken, err := repo.GetEmailVerificationToken(c.Context(), token)
	if err != nil {
		return renderVerifyEmailPage(c, view, "verify.error_invalid_token", "")
	}

	// Check if token is expired
	if time.Now().After(verificationToken.ExpiresAt) {
		return renderVerifyEmailPage(c, view, "verify.error_expired_token", "")
	}

	// Get user
	user, err := repo.GetUserByID(c.Context(), verificationToken.UserID)
	if err != nil {
		return renderVerifyEmailPage(c, view, "verify.error_user_not_found", "")
	}

	// Verify email
	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := repo.UpdateUser(c.Context(), user); err != nil {
		return renderVerifyEmailPage(c, view, "verify.error_verification_failed", "")
	}

	// Delete verification token
	if err := repo.DeleteEmailVerificationToken(c.Context(), token); err != nil {
		log.Printf("Error deleting verification token: %v", err)
	}

	// Render success page
	return renderVerifyEmailPage(c, view, "", "success")
}

// Helper function to render verify email page
func renderVerifyEmailPage(c *fiber.Ctx, view *jet.Set, errorKey, success string) error {
	tmpl, err := view.GetTemplate("pages/verify-email.jet")
	if err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	vars := make(jet.VarMap)
	if errorKey != "" {
		vars.Set("Error", errorKey) // TODO: translate error message
	}
	if success != "" {
		vars.Set("Success", true)
	}
	vars.Set("CsrfToken", c.Locals("csrf"))

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}
	vars.Set("ShowUserMenu", false) // Not authenticated on verify page

	// Add i18n translation function
	if t, ok := c.Locals("t").(func(string, ...interface{}) string); ok {
		vars.Set("t", t)
	}

	return renderTemplate(c, tmpl, vars)
}

// renderTemplate renders a Jet template with variables
func renderTemplate(c *fiber.Ctx, tmpl *jet.Template, vars jet.VarMap) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars, nil); err != nil {
		return err
	}
	c.Type("html")
	return c.Send(buf.Bytes())
}

// userRepositoryWrapper wraps the database repository to implement UserRepository interface
type userRepositoryWrapper struct {
	repo *database.Repository
}

func (w *userRepositoryWrapper) CreateUser(user *models.User) error {
	return w.repo.CreateUser(context.Background(), user)
}

func (w *userRepositoryWrapper) GetUserByEmail(email string) (*models.User, error) {
	return w.repo.GetUserByEmail(context.Background(), email)
}

func (w *userRepositoryWrapper) GetUserByEmailVerificationToken(token string) (*models.User, error) {
	return w.repo.GetUserByEmailVerificationToken(context.Background(), token)
}

func (w *userRepositoryWrapper) UpdateUser(user *models.User) error {
	return w.repo.UpdateUser(context.Background(), user)
}

func (w *userRepositoryWrapper) CreateEmailVerificationToken(token *models.EmailVerificationToken) error {
	return w.repo.CreateEmailVerificationToken(context.Background(), token)
}

func (w *userRepositoryWrapper) GetEmailVerificationToken(token string) (*models.EmailVerificationToken, error) {
	return w.repo.GetEmailVerificationToken(context.Background(), token)
}

func (w *userRepositoryWrapper) UpdateEmailVerificationToken(token *models.EmailVerificationToken) error {
	return w.repo.UpdateEmailVerificationToken(context.Background(), token)
}

// RegisterHandler handles user registration (alias for the existing RegisterUser function)
func RegisterHandler(c *fiber.Ctx) error {
	// This is a simple alias to the existing registration logic
	// We can use the existing RegistrationHandler struct method approach

	view := c.Locals("view").(*jet.Set)
	repo := c.Locals("repo").(*database.Repository)

	// Create services
	emailConfig := email.NewConfigFromEnv()
	emailService := email.NewService(emailConfig)
	passwordService := auth.NewPasswordService()

	// Create a repository wrapper that implements UserRepository interface
	repoWrapper := &userRepositoryWrapper{repo: repo}

	// Get form data
	email := strings.TrimSpace(c.FormValue("email"))
	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	// Basic validation
	if email == "" || password == "" {
		return renderRegisterPage(c, view, "Email and password are required", email, username)
	}

	if password != confirmPassword {
		return renderRegisterPage(c, view, "Passwords do not match", email, username)
	}

	// Validate email format
	if _, err := mail.ParseAddress(email); err != nil {
		return renderRegisterPage(c, view, "Invalid email address", email, username)
	}

	// Validate password strength
	if len(password) < 8 {
		return renderRegisterPage(c, view, "Password must be at least 8 characters long", email, username)
	}

	// Create new user
	user := models.NewEmailUser(email, "")
	if username != "" {
		user.SetUsername(username)
	}

	// Hash password
	passwordHash, err := passwordService.HashPassword(password)
	if err != nil {
		return renderRegisterPage(c, view, "Registration failed. Please try again.", email, username)
	}
	user.PasswordHash = &passwordHash

	// Generate verification token
	token, err := generateSecureToken()
	if err != nil {
		return renderRegisterPage(c, view, "Registration failed. Please try again.", email, username)
	}

	// Set verification token (expires in 24 hours)
	expiresAt := time.Now().Add(24 * time.Hour)

	// Save user
	if err := repoWrapper.CreateUser(user); err != nil {
		return renderRegisterPage(c, view, "Email address is already taken", "", username)
	}

	// Create verification token record
	verificationToken := models.NewEmailVerificationToken(
		user.ID,
		token,
		email,
		expiresAt,
	)

	if err := repoWrapper.CreateEmailVerificationToken(verificationToken); err != nil {
		log.Printf("Error creating verification token: %v", err)
		// Continue anyway, user can request new token later
	}

	// Send verification email
	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "http://localhost:8090" // fallback for development
	}
	verificationURL := fmt.Sprintf("%s/auth/verify-email?token=%s", siteURL, token)

	displayName := username
	if displayName == "" {
		displayName = email
	}

	if err := emailService.SendVerificationEmail(email, displayName, verificationURL); err != nil {
		log.Printf("Error sending verification email: %v", err)
		// Don't fail registration, just show a different message
		return renderRegisterSuccessPage(c, view, email, "Registration successful, but we couldn't send the verification email. Please contact support.")
	}

	return renderRegisterSuccessPage(c, view, email, "")
}

// Helper function to render register page with error
func renderRegisterPage(c *fiber.Ctx, view *jet.Set, errorMsg, email, username string) error {
	tmpl, err := view.GetTemplate("pages/register.jet")
	if err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	vars := make(jet.VarMap)
	vars.Set("Error", errorMsg)
	vars.Set("Email", email)
	vars.Set("Username", username)
	vars.Set("CsrfToken", c.Locals("csrf"))

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}
	vars.Set("ShowUserMenu", false) // Not authenticated on register page

	// Add i18n translation function
	if t, ok := c.Locals("t").(func(string, ...interface{}) string); ok {
		vars.Set("t", t)
	}

	return renderTemplate(c, tmpl, vars)
}

// Helper function to render register success page
func renderRegisterSuccessPage(c *fiber.Ctx, view *jet.Set, email, emailError string) error {
	tmpl, err := view.GetTemplate("pages/register-success.jet")
	if err != nil {
		return c.Status(500).SendString("Internal Server Error")
	}

	vars := make(jet.VarMap)
	vars.Set("Email", email)
	if emailError != "" {
		vars.Set("EmailError", emailError)
	}
	vars.Set("CsrfToken", c.Locals("csrf"))

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}
	vars.Set("ShowUserMenu", false) // Not authenticated on success page

	// Add i18n translation function
	if t, ok := c.Locals("t").(func(string, ...interface{}) string); ok {
		vars.Set("t", t)
	}

	return renderTemplate(c, tmpl, vars)
}
