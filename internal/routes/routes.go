package routes

import (
	"strings"

	"bitcoinpitch.org/internal/auth"
	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/handlers"
	"bitcoinpitch.org/internal/i18n"
	"bitcoinpitch.org/internal/middleware"
	"bitcoinpitch.org/internal/models"

	"context"
	"log"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(app *fiber.App, view *jet.Set, repo *database.Repository, configService *config.Service) {
	// Initialize services
	totpSvc := auth.NewTOTPService("BitcoinPitch.org")

	// Initialize handlers
	totpHandler := handlers.NewTOTPHandler(repo, totpSvc)

	// Initialize admin handler
	adminHandler := handlers.NewAdminHandler(configService, repo)

	// Ensure Jet view is always set in context for every request
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("view", view)
		return c.Next()
	})

	// Set repository in context (Repository wrapper, NOT raw DB)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("repo", repo)
		return c.Next()
	})

	// Set config service in context for all handlers
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("configService", configService)
		return c.Next()
	})

	// DB health check endpoint
	app.Get("/api/health/db", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		if err := repo.Ping(ctx); err != nil {
			log.Printf("Database connection error: %v", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "error",
				"error":  "Database connection error",
			})
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// /api/csrf-token handler - use Fiber's built-in CSRF token
	app.Get("/api/csrf-token", func(c *fiber.Ctx) error {
		// Fiber's CSRF middleware should have set the token in context
		token := c.Locals("token")
		if token == nil {
			log.Printf("/api/csrf-token: No CSRF token found in context")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "CSRF token not available",
			})
		}
		log.Printf("/api/csrf-token: Returning token: %s", token)
		return c.JSON(fiber.Map{
			"csrf_token": token,
		})
	})

	// /api/csrf-test handler
	app.Post("/api/csrf-test", func(c *fiber.Ctx) error {
		log.Println("/api/csrf-test handler called")
		return c.JSON(fiber.Map{"ok": true})
	})

	// Health check endpoints (no auth required)
	app.Get("/health", middleware.HealthCheck)

	// Static files
	app.Static("/static", "./static")

	// Public routes (no auth required)
	public := app.Group("/")
	public.Get("/", handlers.HomeHandler)
	public.Get("/bitcoin", handlers.PitchListHandler)
	public.Get("/lightning", handlers.PitchListHandler)
	public.Get("/cashu", handlers.PitchListHandler)

	// Language switching routes
	app.Get("/lang/:lang", func(c *fiber.Ctx) error {
		// Get the i18n manager from context
		i18nManager := c.Locals("i18nManager")
		if i18nManager == nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "I18n manager not available",
			})
		}

		langCode := c.Params("lang")

		// Validate language code using i18n manager
		manager := i18nManager.(*i18n.Manager)
		availableLanguages := manager.GetAvailableLanguages()
		isValid := false
		for _, lang := range availableLanguages {
			if lang == langCode {
				isValid = true
				break
			}
		}

		if !isValid {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid language code",
			})
		}

		// Set cookie for language preference (30 days)
		c.Cookie(&fiber.Cookie{
			Name:     "bitcoinpitch_lang",
			Value:    langCode,
			MaxAge:   30 * 24 * 60 * 60, // 30 days in seconds
			HTTPOnly: false,             // Allow JS access for UI updates
			Secure:   false,             // Set to true in production with HTTPS
			SameSite: "Lax",
		})

		// Get return URL or default to home
		returnUrl := c.Query("return", "/")
		if returnUrl == "" {
			returnUrl = "/"
		}

		// Redirect back to the page user was on
		return c.Redirect(returnUrl)
	})

	// TEMP: Register /pitch/form as a top-level route to bypass /pitch group middleware
	app.Get("/pitch/form", handlers.PitchFormHandler)

	// Parameterized route MUST come after specific routes
	public.Get("/pitch/:id", handlers.PitchViewHandler)
	public.Get("/p/:id", handlers.PitchShareHandler) // Clean share URL

	// Pitch routes (auth required for create/edit/delete)
	pitches := app.Group("/pitch")
	pitches.Get("/add", handlers.PitchFormHandler)       // Show form
	pitches.Post("/add", handlers.PitchAddHandler)       // Create pitch
	pitches.Get("/:id/edit", handlers.PitchEditHandler)  // Show edit form
	pitches.Post("/:id/edit", handlers.PitchEditHandler) // Update pitch (edit)
	pitches.Delete("/:id", handlers.PitchDeleteHandler)
	pitches.Post("/:id/vote", handlers.PitchVoteHandler) // Vote on pitch (HTMX)
	pitches.Get("/delete-confirm", func(c *fiber.Ctx) error {
		println("[DEBUG] Route matched: /pitch/delete-confirm")
		return handlers.PitchDeleteConfirmHandler(c)
	})
	pitches.Get("/:idOrAny/delete-confirm", func(c *fiber.Ctx) error {
		println("[DEBUG] Route matched: /pitch/:idOrAny/delete-confirm with idOrAny =", c.Params("idOrAny"))
		return handlers.PitchDeleteConfirmHandler(c)
	})
	pitches.Post("/:id/delete", handlers.PitchDeleteHandler)

	// Catch-all for any unmatched /pitch/* route (for modal/HTMX 404s) - MUST BE LAST
	// TEMP: Commented out to test if this is interfering with delete-confirm routes
	// pitches.All("/*", handlers.PitchCatchAllHandler)

	// Authentication routes
	authGroup := app.Group("/auth")
	authGroup.Get("/login", handlers.AuthLoginHandler)
	authGroup.Post("/password", handlers.AuthPasswordHandler)
	authGroup.Post("/trezor", handlers.AuthTrezorHandler)
	authGroup.Post("/nostr", handlers.AuthNostrHandler)
	authGroup.Post("/nostr-manual", handlers.AuthNostrManualHandler)
	// authGroup.Post("/twitter", handlers.AuthTwitterHandler) // DISABLED
	authGroup.Post("/logout", handlers.AuthLogoutHandler)

	// Registration routes
	app.Get("/register", handlers.RegisterPageHandler)
	authGroup.Post("/register", handlers.RegisterHandler)
	authGroup.Get("/verify-email", handlers.VerifyEmailHandler)

	// Search routes
	app.Get("/search", handlers.SearchHandler) // Main search page

	// User routes
	userGroup := app.Group("/user")
	userGroup.Use(middleware.AuthMiddleware(repo))
	userGroup.Use(middleware.RequireAuthMiddleware())
	userGroup.Get("/profile", handlers.UserProfileHandler)
	userGroup.Post("/profile", handlers.UserUpdateHandler)
	userGroup.Post("/privacy", handlers.UserPrivacyHandler)
	userGroup.Post("/pagination", handlers.UserPaginationHandler)
	userGroup.Get("/pitches", func(c *fiber.Ctx) error {
		// Smart redirect for "My Pitches" based on context and user activity

		// Option 1: Check if user came from a specific category page (referer-based)
		referer := c.Get("Referer")
		if referer != "" {
			if strings.Contains(referer, "/lightning") {
				return c.Redirect("/lightning?author=me", fiber.StatusTemporaryRedirect)
			} else if strings.Contains(referer, "/cashu") {
				return c.Redirect("/cashu?author=me", fiber.StatusTemporaryRedirect)
			} else if strings.Contains(referer, "/bitcoin") {
				return c.Redirect("/bitcoin?author=me", fiber.StatusTemporaryRedirect)
			}
		}

		// Option 2: Get user's most active category from database
		user := c.Locals("user").(*models.User)
		repo := c.Locals("repo").(*database.Repository)

		// Query user's pitch count by category to find their most active category
		categories := []string{"bitcoin", "lightning", "cashu"}
		var bestCategory string
		maxCount := 0

		for _, category := range categories {
			filters := map[string]interface{}{
				"main_category": category,
				"user_id":       user.ID,
			}
			pitches, err := repo.ListPitches(c.Context(), filters, 1000, 0) // Get all pitches for counting
			if err == nil && len(pitches) > maxCount {
				maxCount = len(pitches)
				bestCategory = category
			}
		}

		// Option 3: Default to bitcoin if no activity found
		if bestCategory == "" {
			bestCategory = "bitcoin"
		}

		return c.Redirect("/"+bestCategory+"?author=me", fiber.StatusTemporaryRedirect)
	})

	// 2FA routes (must be under userGroup to have authentication middleware)
	userGroup.Post("/2fa/generate", totpHandler.GenerateTOTPSecret)
	userGroup.Get("/2fa/qr", totpHandler.GenerateQRCode)
	userGroup.Post("/2fa/enable", totpHandler.EnableTOTP)
	userGroup.Post("/2fa/disable", totpHandler.DisableTOTP)
	userGroup.Post("/2fa/backup-codes", totpHandler.GetBackupCodes)

	// API routes (for HTMX and other AJAX requests)
	api := app.Group("/api")

	// Health check endpoint
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// Simple test endpoint to verify route registration
	api.Get("/test", func(c *fiber.Ctx) error {
		println("[DEBUG] Test endpoint called successfully")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Route handlers are working",
		})
	})
	api.Get("/pitches", handlers.APIPitchesListHandler)
	api.Get("/pitches/:id", handlers.APIPitchGetHandler)
	api.Post("/pitches", handlers.APIPitchCreateHandler)
	api.Put("/pitches/:id", handlers.APIPitchUpdateHandler)
	api.Delete("/pitches/:id", handlers.APIPitchDeleteHandler)
	api.Post("/pitches/:id/vote", handlers.APIPitchVoteHandler)

	// Tag routes
	api.Get("/tags/suggestions", handlers.TagSuggestionsHandler)
	api.Get("/tags", handlers.TagListHandler)

	// Configuration routes
	api.Get("/config/pitch-limits", func(c *fiber.Ctx) error {
		configService := c.Locals("configService").(*config.Service)
		limits := configService.PitchLimits(c.Context())
		return c.JSON(limits)
	})

	// Language routes
	api.Get("/languages/usage", handlers.APILanguageUsageHandler)

	// Search routes
	api.Get("/search", handlers.APISearchHandler)

	// Admin routes (require admin role)
	log.Println("[DEBUG] Setting up admin routes...")
	adminRoutes := app.Group("/admin")
	// First apply auth middleware to populate user in context
	adminRoutes.Use(middleware.AuthMiddleware(repo))
	adminRoutes.Use(middleware.RequireAuthMiddleware())
	// Then check for admin role
	adminRoutes.Use(func(c *fiber.Ctx) error {
		user := c.Locals("user").(*models.User)
		log.Printf("[DEBUG] Admin middleware: user=%v, role=%v", user.GetDisplayName(), user.Role)
		if user == nil || user.Role != models.UserRoleAdmin {
			log.Printf("[DEBUG] Admin access denied: user role=%v, required=%v", user.Role, models.UserRoleAdmin)
			return c.Status(fiber.StatusForbidden).SendString("Admin access required")
		}
		log.Println("[DEBUG] Admin access granted")
		return c.Next()
	})

	// Admin dashboard and management
	log.Println("[DEBUG] Registering admin routes...")
	adminRoutes.Get("/", adminHandler.AdminDashboardHandler)
	adminRoutes.Get("/config", adminHandler.AdminConfigHandler)
	adminRoutes.Post("/config", adminHandler.AdminConfigUpdateHandler)
	adminRoutes.Get("/users", adminHandler.AdminUsersHandler)
	adminRoutes.Post("/users/:id/role", adminHandler.AdminUserUpdateRoleHandler)
	adminRoutes.Post("/users/:id/disable", adminHandler.AdminUserDisableHandler)
	adminRoutes.Post("/users/:id/hide", adminHandler.AdminUserHideHandler)
	adminRoutes.Post("/users/:id/delete", adminHandler.AdminUserDeleteHandler)
	adminRoutes.Get("/pitches", adminHandler.AdminPitchesHandler)
	adminRoutes.Post("/pitches/:id/delete", adminHandler.AdminPitchDeleteHandler)
	adminRoutes.Post("/pitches/:id/hide", adminHandler.AdminPitchHideHandler)
	adminRoutes.Get("/audit-logs", adminHandler.AdminAuditLogsHandler)
	log.Println("[DEBUG] Admin routes registered successfully")

	// Error handlers
	app.Use(handlers.NotFoundHandler) // Global 404 handler (HTMX/modal aware)
	// app.Use(handlers.ErrorHandler)    // 500 handler (not implemented)
}
