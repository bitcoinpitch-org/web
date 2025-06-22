package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"bitcoinpitch.org/internal/antispam"
	"bitcoinpitch.org/internal/auth"
	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/i18n"
	"bitcoinpitch.org/internal/middleware"
	"bitcoinpitch.org/internal/models"
	"bitcoinpitch.org/internal/routes"
	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

// Dummy data for testing
var dummyPitches = []map[string]interface{}{
	{
		"ID":           "1",
		"Content":      "Bitcoin is digital gold that you can send anywhere in the world.",
		"PostedBy":     "Satoshi",
		"AuthorType":   "same",
		"MainCategory": "bitcoin",
		"Tags":         []string{"gold", "digital", "global"},
		"Upvotes":      42,
		"Downvotes":    3,
		"Score":        39,
	},
	{
		"ID":           "2",
		"Content":      "Lightning Network makes Bitcoin payments instant and nearly free.",
		"PostedBy":     "Alice",
		"AuthorType":   "custom",
		"AuthorName":   "Bitcoin Enthusiast",
		"MainCategory": "lightning",
		"Tags":         []string{"instant", "payments", "scaling"},
		"Upvotes":      35,
		"Downvotes":    2,
		"Score":        33,
	},
	{
		"ID":           "3",
		"Content":      "Cashu is a privacy-focused Bitcoin ecash system.",
		"PostedBy":     "Bob",
		"AuthorType":   "twitter",
		"AuthorHandle": "@bitcoinbob",
		"MainCategory": "cashu",
		"Tags":         []string{"privacy", "ecash", "bitcoin"},
		"Upvotes":      28,
		"Downvotes":    1,
		"Score":        27,
	},
}

func dict(args jet.Arguments) reflect.Value {
	m := make(map[string]interface{})
	for i := 0; i < args.NumOfArguments(); i += 2 {
		key := args.Get(i).String()
		val := args.Get(i + 1).Interface()
		m[key] = val
	}
	return reflect.ValueOf(m)
}

func trimPrefix(args jet.Arguments) reflect.Value {
	args.RequireNumOfArguments("trimPrefix", 2, 2)
	str := args.Get(0).String()
	prefix := args.Get(1).String()
	return reflect.ValueOf(strings.TrimPrefix(str, prefix))
}

func length(args jet.Arguments) reflect.Value {
	args.RequireNumOfArguments("length", 1, 1)
	v := args.Get(0)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return reflect.ValueOf(v.Len())
	}
	return reflect.ValueOf(0)
}

func join(args jet.Arguments) reflect.Value {
	args.RequireNumOfArguments("join", 2, 2)
	slice := args.Get(0)
	sep := args.Get(1).String()

	if slice.Kind() != reflect.Slice && slice.Kind() != reflect.Array {
		return reflect.ValueOf("")
	}

	var result []string
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Struct {
			// If it's a struct, try to get the Name field
			if nameField := item.FieldByName("Name"); nameField.IsValid() {
				result = append(result, nameField.String())
			}
		} else {
			result = append(result, item.String())
		}
	}
	return reflect.ValueOf(strings.Join(result, sep))
}

func renderTemplate(c *fiber.Ctx, tmpl *jet.Template, vars jet.VarMap) error {
	// Add common variables to all templates
	vars.Set("now", time.Now())

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars, nil); err != nil {
		return err
	}
	c.Type("html") // Ensure browser renders as HTML
	return c.Send(buf.Bytes())
}

// adminRepositoryWrapper wraps the repository to match the AdminRepository interface
type adminRepositoryWrapper struct {
	repo *database.Repository
}

func (w *adminRepositoryWrapper) CreateUser(user *models.User) error {
	return w.repo.CreateUser(context.Background(), user)
}

func (w *adminRepositoryWrapper) GetUserByEmail(email string) (*models.User, error) {
	return w.repo.GetUserByEmail(context.Background(), email)
}

func (w *adminRepositoryWrapper) GetUserByRole(role models.UserRole) ([]*models.User, error) {
	return w.repo.GetUserByRole(context.Background(), role)
}

func (w *adminRepositoryWrapper) UpdateUser(user *models.User) error {
	return w.repo.UpdateUser(context.Background(), user)
}

func (w *adminRepositoryWrapper) CountUsersByRole(role models.UserRole) (int, error) {
	return w.repo.CountUsersByRole(context.Background(), role)
}

func main() {
	log.Println("Starting server...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize database connection
	dbCfg := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     5432, // Default PostgreSQL port
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  "disable", // For local development
	}

	db, err := database.New(dbCfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Run database migrations automatically
	log.Println("Running database migrations...")
	if err := database.RunMigrations(dbCfg, "/app/migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize admin user if ADMIN_SETUP_TOKEN is provided
	log.Println("Checking for admin user initialization...")
	repo := database.NewRepository(db)
	passwordSvc := auth.NewPasswordService()
	totpSvc := auth.NewTOTPService("BitcoinPitch.org")

	// Create compatibility wrapper for admin repository
	adminRepo := &adminRepositoryWrapper{repo: repo}
	adminSvc := auth.NewAdminService(adminRepo, passwordSvc, totpSvc)

	if err := adminSvc.InitializeAdminUser(); err != nil {
		log.Printf("Warning: Failed to initialize admin user: %v", err)
	}

	// Initialize authentication providers
	log.Println("Initializing authentication providers...")

	// Initialize Twitter OAuth
	auth.InitTwitterOAuth()
	if err := auth.ValidateTwitterConfig(); err != nil {
		log.Printf("Warning: Twitter OAuth not properly configured: %v", err)
		log.Println("Twitter authentication will be disabled")
	} else {
		log.Println("Twitter OAuth initialized successfully")
	}

	// Initialize configuration service
	log.Println("Initializing configuration service...")
	configService := config.NewService(repo)
	if err := configService.RefreshCache(context.Background()); err != nil {
		log.Printf("Warning: Failed to load configuration cache: %v", err)
		log.Println("Configuration service will use defaults")
	} else {
		log.Println("Configuration service initialized successfully")
	}

	// Initialize antispam service
	log.Println("Initializing antispam service...")
	antispamService := antispam.NewService(repo, configService)
	log.Println("Antispam service initialized successfully")

	// Initialize internationalization
	log.Println("Initializing i18n system...")
	i18nManager := i18n.NewManager("en") // Default to English
	if err := i18nManager.LoadTranslations("/app/i18n"); err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}
	log.Println("I18n system initialized successfully")

	// Create Jet template engine
	view := jet.NewSet(
		jet.NewOSFileSystemLoader("/app/templates"),
		jet.InDevelopmentMode(), // Enable development mode for better error messages
	)

	// Add global functions
	view.AddGlobalFunc("dict", dict)
	view.AddGlobalFunc("now", func(args jet.Arguments) reflect.Value {
		return reflect.ValueOf(time.Now())
	})
	view.AddGlobalFunc("trimPrefix", trimPrefix)
	view.AddGlobalFunc("length", length)
	view.AddGlobalFunc("join", join)
	view.AddGlobalFunc("formatDate", func(args jet.Arguments) reflect.Value {
		if args.NumOfArguments() < 2 {
			return reflect.ValueOf("")
		}

		if timeArg := args.Get(0); timeArg.IsValid() {
			if formatArg := args.Get(1); formatArg.IsValid() {
				if t, ok := timeArg.Interface().(time.Time); ok {
					if format, ok := formatArg.Interface().(string); ok {
						return reflect.ValueOf(t.Format(format))
					}
				}
			}
		}
		return reflect.ValueOf("")
	})

	// Add translation helper function
	view.AddGlobalFunc("t", func(args jet.Arguments) reflect.Value {
		// Get current language from context (set by i18n middleware)
		lang := "en" // default
		if args.NumOfArguments() > 1 {
			if langArg := args.Get(1); langArg.IsValid() {
				if s, ok := langArg.Interface().(string); ok {
					lang = s
				}
			}
		}

		// Get translation key
		if args.NumOfArguments() > 0 {
			if keyArg := args.Get(0); keyArg.IsValid() {
				if key, ok := keyArg.Interface().(string); ok {
					// Get translation from i18n manager
					if translation := i18nManager.GetTranslation(lang, key); translation != "" {
						return reflect.ValueOf(translation)
					}
					// Fallback to key if translation not found
					return reflect.ValueOf(key)
				}
			}
		}
		return reflect.ValueOf("")
	})

	// Add pagination URL builder function
	view.AddGlobalFunc("buildPaginationURL", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("buildPaginationURL", 2, 2)
		pageArg := args.Get(0)
		sizeArg := args.Get(1)

		var page, size int

		// Handle different numeric types
		if pageArg.Kind() == reflect.Float64 {
			page = int(pageArg.Float())
		} else {
			page = int(pageArg.Int())
		}

		if sizeArg.Kind() == reflect.Float64 {
			size = int(sizeArg.Float())
		} else {
			size = int(sizeArg.Int())
		}

		// Return a placeholder that JavaScript will replace with proper URL
		// This ensures all existing query parameters are preserved
		return reflect.ValueOf(fmt.Sprintf("javascript:goToPage(%d,%d)", page, size))
	})

	// Add iter function for creating page number ranges
	view.AddGlobalFunc("iter", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("iter", 2, 2)
		start := args.Get(0)
		end := args.Get(1)

		var startInt, endInt int

		// Handle different numeric types
		if start.Kind() == reflect.Float64 {
			startInt = int(start.Float())
		} else {
			startInt = int(start.Int())
		}

		if end.Kind() == reflect.Float64 {
			endInt = int(end.Float())
		} else {
			endInt = int(end.Int())
		}

		if startInt > endInt {
			return reflect.ValueOf([]int{})
		}

		result := make([]int, endInt-startInt+1)
		for i := 0; i < len(result); i++ {
			result[i] = startInt + i
		}
		return reflect.ValueOf(result)
	})

	// Add max function
	view.AddGlobalFunc("max", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("max", 2, 2)
		a := args.Get(0)
		b := args.Get(1)

		var aInt, bInt int

		if a.Kind() == reflect.Float64 {
			aInt = int(a.Float())
		} else {
			aInt = int(a.Int())
		}

		if b.Kind() == reflect.Float64 {
			bInt = int(b.Float())
		} else {
			bInt = int(b.Int())
		}

		if aInt > bInt {
			return reflect.ValueOf(aInt)
		}
		return reflect.ValueOf(bInt)
	})

	// Add min function
	view.AddGlobalFunc("min", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("min", 2, 2)
		a := args.Get(0)
		b := args.Get(1)

		var aInt, bInt int

		if a.Kind() == reflect.Float64 {
			aInt = int(a.Float())
		} else {
			aInt = int(a.Int())
		}

		if b.Kind() == reflect.Float64 {
			bInt = int(b.Float())
		} else {
			bInt = int(b.Int())
		}

		if aInt < bInt {
			return reflect.ValueOf(aInt)
		}
		return reflect.ValueOf(bInt)
	})

	// Add arithmetic functions for template safety
	view.AddGlobalFunc("add", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("add", 2, 2)
		a := args.Get(0)
		b := args.Get(1)

		var aInt, bInt int

		if a.Kind() == reflect.Float64 {
			aInt = int(a.Float())
		} else {
			aInt = int(a.Int())
		}

		if b.Kind() == reflect.Float64 {
			bInt = int(b.Float())
		} else {
			bInt = int(b.Int())
		}

		return reflect.ValueOf(aInt + bInt)
	})

	view.AddGlobalFunc("sub", func(args jet.Arguments) reflect.Value {
		args.RequireNumOfArguments("sub", 2, 2)
		a := args.Get(0)
		b := args.Get(1)

		var aInt, bInt int

		if a.Kind() == reflect.Float64 {
			aInt = int(a.Float())
		} else {
			aInt = int(a.Int())
		}

		if b.Kind() == reflect.Float64 {
			bInt = int(b.Float())
		} else {
			bInt = int(b.Int())
		}

		return reflect.ValueOf(aInt - bInt)
	})

	// Test template loading
	if _, err := view.GetTemplate("layouts/base.jet"); err != nil {
		log.Fatalf("Failed to load base template: %v", err)
	} else {
		log.Printf("Successfully loaded base template")
	}

	log.Println("Template engine initialized")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("Error occurred: %v", err)
			// Return 403 for Forbidden errors
			if errors.Is(err, fiber.ErrForbidden) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Forbidden",
				})
			}
			if strings.HasPrefix(c.Path(), "/api/") {
				// Return JSON for API endpoints
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}
			// Get the Jet view from the context
			view := c.Locals("view").(*jet.Set)

			// Create template variables
			vars := make(jet.VarMap)
			vars.Set("Title", "Server Error")
			vars.Set("Error", err.Error())
			// Pass CSRF token to template
			if csrfToken := c.Locals("csrf"); csrfToken != nil {
				vars.Set("CsrfToken", csrfToken)
			}

			// Render the 500 template
			t, err := view.GetTemplate("pages/500.jet")
			if err != nil {
				return c.SendString("TEMPLATE ERROR: " + err.Error())
			}

			if err := t.Execute(c.Response().BodyWriter(), vars, nil); err != nil {
				return c.SendString("TEMPLATE EXECUTION ERROR: " + err.Error())
			}

			return nil
		},
	})

	// Add logger middleware FIRST
	app.Use(logger.New())

	// Re-enable only the Jet view context middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("view", view)
		return c.Next()
	})

	// Add i18n middleware
	app.Use(middleware.I18n(middleware.I18nConfig{
		I18nManager: i18nManager,
		CookieName:  "bitcoinpitch_lang",
		DefaultLang: "en",
	}))

	// DO NOT store repository in context - it causes DB connection closure
	// Repository is passed directly to handlers that need it

	// Add authentication middleware
	app.Use(middleware.AuthMiddleware(repo))

	// Add antispam middleware
	app.Use(middleware.AntiSpamMiddleware(antispamService))

	// Re-enable static file serving
	app.Static("/static", "./static")

	// Re-enable security middleware
	middleware.SecurityMiddleware(app)

	// Setup all routes from routes package (handles all routing including 404)
	routes.SetupRoutes(app, view, repo, configService)

	// Start server
	log.Println("Server starting on :8090")
	log.Fatal(app.Listen(":8090"))
}
