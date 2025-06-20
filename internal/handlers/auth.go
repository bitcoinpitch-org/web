package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"bitcoinpitch.org/internal/auth"
	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/crypto"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/middleware"
	"bitcoinpitch.org/internal/models"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
)

// AuthLoginHandler renders the login modal
func AuthLoginHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	vars := make(jet.VarMap)
	vars.Set("Title", "Login")

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	t, err := view.GetTemplate("partials/auth-login.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// AuthTrezorHandler handles Trezor hardware wallet authentication
func AuthTrezorHandler(c *fiber.Ctx) error {
	var req struct {
		Message   string `json:"message"`
		Signature string `json:"signature"`
		Address   string `json:"address"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Message == "" || req.Signature == "" || req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields",
		})
	}

	// Validate Bitcoin address format
	if err := crypto.ValidateBitcoinAddress(req.Address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Bitcoin address",
		})
	}

	// Verify the Bitcoin message signature
	if err := crypto.VerifyBitcoinMessage(req.Message, req.Signature, req.Address); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature: " + err.Error(),
		})
	}

	repo := c.Locals("repo").(*database.Repository)

	// Check if user already exists
	user, err := repo.GetUserByAuth(c.Context(), models.AuthTypeTrezor, req.Address)
	if err != nil {
		// User doesn't exist, create new user
		user = models.NewUser(models.AuthTypeTrezor, req.Address)
		user.SetDisplayName("Trezor User")

		if err := repo.CreateUser(c.Context(), user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user: " + err.Error(),
			})
		}
	}

	// Create session
	token, err := middleware.CreateSession(repo, c.Context(), user.BaseModel.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session: " + err.Error(),
		})
	}

	// Set session cookie
	middleware.SetSessionCookie(c, token)

	return c.JSON(fiber.Map{
		"message": "Authentication successful",
		"user":    user.GetDisplayName(),
	})
}

// AuthNostrHandler handles Nostr authentication
func AuthNostrHandler(c *fiber.Ctx) error {
	var req struct {
		Event   map[string]interface{} `json:"event"`
		Message string                 `json:"message"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate that event is provided
	if req.Event == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing event",
		})
	}

	// Extract and validate pubkey from event
	pubkey, err := crypto.ExtractPubkeyFromEvent(req.Event)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid event pubkey: " + err.Error(),
		})
	}

	// Verify the Nostr event signature
	if err := crypto.VerifyNostrEvent(req.Event); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Nostr signature: " + err.Error(),
		})
	}

	repo := c.Locals("repo").(*database.Repository)

	// Check if user already exists
	user, err := repo.GetUserByAuth(c.Context(), models.AuthTypeNostr, pubkey)
	if err != nil {
		// User doesn't exist, create new user
		user = models.NewUser(models.AuthTypeNostr, pubkey)

		// Generate a unique username and display name from the pubkey
		username := crypto.GenerateNostrUsername(pubkey)
		displayName := crypto.GenerateNostrDisplayName(pubkey)

		user.SetUsername(username)
		user.SetDisplayName(displayName)

		if err := repo.CreateUser(c.Context(), user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user: " + err.Error(),
			})
		}
	}

	// Create session
	token, err := middleware.CreateSession(repo, c.Context(), user.BaseModel.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session: " + err.Error(),
		})
	}

	// Set session cookie
	middleware.SetSessionCookie(c, token)

	return c.JSON(fiber.Map{
		"message": "Authentication successful",
		"user":    user.GetDisplayName(),
	})
}

// AuthTwitterHandler initiates Twitter OAuth flow
func AuthTwitterHandler(c *fiber.Ctx) error {
	// Validate Twitter OAuth configuration
	if err := auth.ValidateTwitterConfig(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Twitter authentication not configured: " + err.Error(),
		})
	}

	// Generate a random state parameter for CSRF protection
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate state parameter",
		})
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in session for verification in callback
	c.Cookie(&fiber.Cookie{
		Name:     "twitter_oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   600, // 10 minutes
	})

	// Get the Twitter OAuth URL
	authURL := auth.GetTwitterAuthURL(state)

	// Redirect to Twitter OAuth
	return c.Redirect(authURL)
}

// AuthPasswordHandler handles email/password authentication with 2FA support
func AuthPasswordHandler(c *fiber.Ctx) error {
	log.Printf("[DEBUG] AuthPasswordHandler: Starting password authentication")

	var req struct {
		Username string `form:"username"`
		Password string `form:"password"`
		TOTPCode string `form:"totp_code"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[DEBUG] AuthPasswordHandler: Body parsing failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("[DEBUG] AuthPasswordHandler: Username: %s, Has password: %t, Has TOTP: %t",
		req.Username, req.Password != "", req.TOTPCode != "")

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		log.Printf("[DEBUG] AuthPasswordHandler: Missing username or password")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	repo := c.Locals("repo").(*database.Repository)
	passwordSvc := auth.NewPasswordService()
	totpSvc := auth.NewTOTPService("BitcoinPitch.org")

	// Get user by email
	log.Printf("[DEBUG] AuthPasswordHandler: Looking up user by email: %s", req.Username)
	user, err := repo.GetUserByEmail(c.Context(), req.Username)
	if err != nil {
		log.Printf("[DEBUG] AuthPasswordHandler: User lookup failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	log.Printf("[DEBUG] AuthPasswordHandler: User found: %s, Email verified: %t, TOTP enabled: %t",
		user.GetDisplayName(), user.EmailVerified, user.TOTPEnabled)

	// Verify password
	if user.PasswordHash == nil || !passwordSvc.VerifyPassword(req.Password, *user.PasswordHash) {
		log.Printf("[DEBUG] AuthPasswordHandler: Password verification failed - Hash exists: %t", user.PasswordHash != nil)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	log.Printf("[DEBUG] AuthPasswordHandler: Password verification successful")

	// Check if email is verified
	if !user.EmailVerified {
		log.Printf("[DEBUG] AuthPasswordHandler: Email not verified")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Email address not verified. Please check your email for verification link.",
		})
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			log.Printf("[DEBUG] AuthPasswordHandler: 2FA required, no code provided")
			// Return special response indicating 2FA is required
			return c.JSON(fiber.Map{
				"requires_2fa": true,
				"message":      "Two-factor authentication required",
			})
		}

		// Validate TOTP code
		if user.TOTPSecret == nil {
			log.Printf("[DEBUG] AuthPasswordHandler: 2FA enabled but no secret found")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "2FA configuration error",
			})
		}

		isValid := totpSvc.ValidateCode(*user.TOTPSecret, req.TOTPCode)
		log.Printf("[DEBUG] AuthPasswordHandler: TOTP validation result: %t", isValid)
		if !isValid {
			// Check backup codes
			backupCodeValid := false
			if len(user.TOTPBackupCodes) > 0 {
				for i, code := range user.TOTPBackupCodes {
					if code == req.TOTPCode {
						// Remove used backup code
						user.TOTPBackupCodes = append(user.TOTPBackupCodes[:i], user.TOTPBackupCodes[i+1:]...)
						if err := repo.UpdateUser(c.Context(), user); err != nil {
							log.Printf("Error updating user backup codes: %v", err)
						}
						backupCodeValid = true
						break
					}
				}
			}

			if !backupCodeValid {
				log.Printf("[DEBUG] AuthPasswordHandler: 2FA code validation failed")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid 2FA code",
				})
			}
		}
	}

	// Create session
	token, err := middleware.CreateSession(repo, c.Context(), user.BaseModel.ID)
	if err != nil {
		log.Printf("[DEBUG] AuthPasswordHandler: Session creation failed: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session: " + err.Error(),
		})
	}

	// Set session cookie
	middleware.SetSessionCookie(c, token)

	log.Printf("[DEBUG] AuthPasswordHandler: Authentication successful for user: %s", user.GetDisplayName())

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		// For HTMX requests, use HX-Redirect to force a page redirect
		c.Set("HX-Redirect", "/")
		return c.SendString(`<div class="auth-success">Authentication successful! Redirecting...</div>`)
	}

	return c.JSON(fiber.Map{
		"message": "Authentication successful",
		"user":    user.GetDisplayName(),
	})
}

// AuthLogoutHandler handles user logout
func AuthLogoutHandler(c *fiber.Ctx) error {
	// Get session token from cookie
	sessionToken := c.Cookies("session_token")
	if sessionToken != "" {
		repo := c.Locals("repo").(*database.Repository)

		// Get session by token
		session, err := repo.GetSessionByToken(c.Context(), sessionToken)
		if err == nil {
			// Delete session from database
			repo.DeleteSession(c.Context(), session.BaseModel.ID)
		}

		// Clear session cookie
		c.ClearCookie("session_token")
	}

	// Redirect to home page
	return c.Redirect("/")
}

// AuthCallbackHandler handles OAuth callbacks
func AuthCallbackHandler(c *fiber.Ctx) error {
	provider := c.Params("provider")

	switch provider {
	case "twitter":
		return handleTwitterCallback(c)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unsupported OAuth provider",
		})
	}
}

// handleTwitterCallback handles Twitter OAuth callback
func handleTwitterCallback(c *fiber.Ctx) error {
	// Get the authorization code from query parameters
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Authorization code not provided",
		})
	}

	// Get and verify the state parameter
	state := c.Query("state")
	storedState := c.Cookies("twitter_oauth_state")
	if state == "" || storedState == "" || state != storedState {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid state parameter",
		})
	}

	// Clear the state cookie
	c.ClearCookie("twitter_oauth_state")

	// Exchange the authorization code for an access token
	token, err := auth.ExchangeCodeForToken(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange token: " + err.Error(),
		})
	}

	// Get user info from Twitter API
	twitterUser, err := auth.GetTwitterUserInfo(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user info: " + err.Error(),
		})
	}

	repo := c.Locals("repo").(*database.Repository)

	// Check if user already exists (use Twitter ID as auth_id)
	user, err := repo.GetUserByAuth(c.Context(), models.AuthTypeTwitter, twitterUser.ID)
	if err != nil {
		// User doesn't exist, create new user
		user = models.NewUser(models.AuthTypeTwitter, twitterUser.ID)
		user.SetUsername(twitterUser.Username)
		user.SetDisplayName(twitterUser.Name)

		if err := repo.CreateUser(c.Context(), user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user: " + err.Error(),
			})
		}
	}

	// Create session
	sessionToken, err := middleware.CreateSession(repo, c.Context(), user.BaseModel.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session: " + err.Error(),
		})
	}

	// Set session cookie
	middleware.SetSessionCookie(c, sessionToken)

	// Redirect to home page with success
	return c.Redirect("/")
}

// UserProfileHandler renders the user profile page
func UserProfileHandler(c *fiber.Ctx) error {
	// FORCE DEBUG - this should appear in logs
	log.Println("=== UserProfileHandler CALLED ===")

	view := c.Locals("view").(*jet.Set)
	vars := make(jet.VarMap)
	vars.Set("Title", "User Profile")

	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	println("[DEBUG] UserProfileHandler: user found in context:", ok)
	if user != nil {
		println("[DEBUG] UserProfileHandler: user ID:", user.ID.String())
		println("[DEBUG] UserProfileHandler: user display name:", user.GetDisplayName())
	} else {
		println("[DEBUG] UserProfileHandler: user is nil")
	}
	if !ok || user == nil {
		println("[DEBUG] UserProfileHandler: redirecting to login")
		return c.Redirect("/auth/login")
	}

	vars.Set("User", user)
	vars.Set("UserDisplayName", user.GetDisplayName())
	vars.Set("AuthStatus", "authenticated")
	vars.Set("ShowUserMenu", true)

	// Debug logging
	log.Printf("[DEBUG] UserProfileHandler: Setting UserDisplayName to: %q", user.GetDisplayName())
	log.Printf("[DEBUG] UserProfileHandler: ShowUserMenu set to: %v", true)

	// Fetch user statistics
	repo := c.Locals("repo").(*database.Repository)
	configService := c.Locals("configService").(*config.Service)

	// Get pagination configuration
	paginationConfig := configService.PaginationConfig(c.Context())
	vars.Set("PaginationConfig", paginationConfig)

	// Get user's pitch count and total score
	pitches, err := repo.ListPitches(c.Context(), map[string]interface{}{
		"user_id": user.ID,
	}, 1000, 0) // Get up to 1000 pitches to count them
	if err != nil {
		log.Printf("Failed to fetch user pitches for stats: %v", err)
		vars.Set("PitchCount", 0)
		vars.Set("TotalScore", 0)
		vars.Set("VoteCount", 0)
	} else {
		pitchCount := len(pitches)
		totalScore := 0
		for _, pitch := range pitches {
			totalScore += pitch.Score
		}
		vars.Set("PitchCount", pitchCount)
		vars.Set("TotalScore", totalScore)
		// For now, set vote count to 0 since we don't have a direct method
		// TODO: Add GetUserVoteCount method to repository
		vars.Set("VoteCount", 0)

		// Debug logging for statistics
		log.Printf("[DEBUG] UserProfileHandler: PitchCount=%d, TotalScore=%d, VoteCount=%d", pitchCount, totalScore, 0)
	}

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	// Add footer configuration
	addFooterConfig(c, vars)

	t, err := view.GetTemplate("pages/user-profile.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// UserUpdateHandler handles updating user profile
func UserUpdateHandler(c *fiber.Ctx) error {
	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req struct {
		Username    string `form:"username"`
		DisplayName string `form:"display_name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update user fields
	if req.Username != "" {
		user.SetUsername(req.Username)
	}
	if req.DisplayName != "" {
		user.SetDisplayName(req.DisplayName)
	}

	// Save to database
	repo := c.Locals("repo").(*database.Repository)
	if err := repo.UpdateUser(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"user":    user.GetDisplayName(),
	})
}

// UserPitchesHandler renders the user's pitches
func UserPitchesHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	vars := make(jet.VarMap)
	vars.Set("Title", "My Pitches")

	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Redirect("/auth/login")
	}

	vars.Set("User", user)
	vars.Set("UserDisplayName", user.GetDisplayName())
	vars.Set("AuthStatus", "authenticated")
	vars.Set("ShowUserMenu", true)

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}

	// Load user's pitches from database
	repo := c.Locals("repo").(*database.Repository)
	pitches, err := repo.ListPitches(c.Context(), map[string]interface{}{
		"user_id": user.ID,
	}, 100, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load user pitches: " + err.Error())
	}

	vars.Set("Pitches", pitches)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	// Add footer configuration
	addFooterConfig(c, vars)

	t, err := view.GetTemplate("pages/user-pitches.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// UserDisplayNameHandler handles updating user display name
func UserDisplayNameHandler(c *fiber.Ctx) error {
	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Redirect("/auth/login")
	}

	// Parse form data
	displayName := c.FormValue("display_name")
	if displayName == "" {
		// TODO: Add flash message for error
		return c.Redirect("/user/profile?error=display_name_required")
	}

	// Validate display name length
	if len(displayName) > 50 {
		// TODO: Add flash message for error
		return c.Redirect("/user/profile?error=display_name_too_long")
	}

	// Update display name
	user.SetDisplayName(displayName)

	// Save to database
	repo := c.Locals("repo").(*database.Repository)
	if err := repo.UpdateUser(c.Context(), user); err != nil {
		log.Printf("Failed to update user display name: %v", err)
		// TODO: Add flash message for error
		return c.Redirect("/user/profile?error=update_failed")
	}

	// TODO: Add flash message for success
	return c.Redirect("/user/profile?success=display_name_updated")
}

// UserPrivacyHandler handles updating user privacy settings
func UserPrivacyHandler(c *fiber.Ctx) error {
	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req struct {
		ShowAuthMethod  bool `form:"show_auth_method"`
		ShowUsername    bool `form:"show_username"`
		ShowProfileInfo bool `form:"show_profile_info"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update privacy settings
	user.SetPrivacySettings(req.ShowAuthMethod, req.ShowUsername, req.ShowProfileInfo)

	// Save to database
	repo := c.Locals("repo").(*database.Repository)
	if err := repo.UpdateUser(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update privacy settings: " + err.Error(),
		})
	}

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		// Return the updated privacy settings section with success feedback
		view := c.Locals("view").(*jet.Set)
		vars := make(jet.VarMap)

		// Set up variables for template
		vars.Set("User", user)
		vars.Set("CsrfToken", c.Locals("csrf"))

		// Set current language from i18n middleware
		if currentLang := c.Locals("currentLang"); currentLang != nil {
			vars.Set("currentLang", currentLang)
		} else {
			vars.Set("currentLang", "en") // fallback to English
		}

		// Add success message flag
		vars.Set("PrivacyUpdateSuccess", true)

		t, err := view.GetTemplate("partials/privacy-settings.jet")
		if err != nil {
			// Fallback to simple success message if template doesn't exist
			return c.SendString(`
				<div id="privacy-settings-container">
					<div class="privacy-success">
						Privacy settings updated successfully!
					</div>
				</div>
			`)
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, vars, nil); err != nil {
			// Fallback to simple success message if template execution fails
			return c.SendString(`
				<div id="privacy-settings-container">
					<div class="privacy-success">
						Privacy settings updated successfully!
					</div>
				</div>
			`)
		}

		return c.Type("html").Send(buf.Bytes())
	}

	return c.JSON(fiber.Map{
		"message": "Privacy settings updated successfully",
	})
}

// UserPaginationHandler handles updating user pagination preferences
func UserPaginationHandler(c *fiber.Ctx) error {
	// Get user from context
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	var req struct {
		PageSize int `form:"page_size"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get pagination configuration to validate page size
	configService := c.Locals("configService").(*config.Service)
	paginationConfig := configService.PaginationConfig(c.Context())

	// Validate page size
	if req.PageSize < 1 || req.PageSize > paginationConfig.MaxPageSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Page size must be between 1 and %d", paginationConfig.MaxPageSize),
		})
	}

	// Check if the page size is in allowed options
	validPageSize := false
	for _, size := range paginationConfig.PageSizeOptions {
		if size == req.PageSize {
			validPageSize = true
			break
		}
	}
	if !validPageSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid page size option",
		})
	}

	// Update page size preference
	user.SetPageSize(req.PageSize)

	// Save to database
	repo := c.Locals("repo").(*database.Repository)
	if err := repo.UpdateUser(c.Context(), user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update pagination settings: " + err.Error(),
		})
	}

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		// Return success message for HTMX
		return c.Type("html").SendString(`
			<div class="pagination-success">
				Pagination preference updated successfully!
			</div>
		`)
	}

	return c.JSON(fiber.Map{
		"message":   "Pagination preference updated successfully",
		"page_size": req.PageSize,
	})
}

// Helper functions

// generatePasswordAuthID creates a consistent auth ID for password authentication
func generatePasswordAuthID(username, password string) string {
	// TODO: Implement proper password hashing (bcrypt, scrypt, etc.)
	// For development, we'll use a simple hash
	hash := sha256.Sum256([]byte(username + ":" + password))
	return hex.EncodeToString(hash[:])
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
