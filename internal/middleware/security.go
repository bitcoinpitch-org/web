package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Error generating CSRF token: %v", err)
		return "fallback-token"
	}
	return hex.EncodeToString(bytes)
}

// GenerateSecureKey generates a secure random key for CSRF
func GenerateSecureKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// If we can't generate a secure key, we should panic as this is a security requirement
		panic("failed to generate secure CSRF key: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// getCORSOrigins gets allowed origins from environment variable
func getCORSOrigins() []string {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if origins == "" {
		// Default to localhost for development
		return []string{"http://localhost:8090"}
	}
	return strings.Split(origins, ",")
}

// SecurityMiddleware sets up all security-related middleware
func SecurityMiddleware(app *fiber.App) {
	// Recover from panics
	app.Use(recover.New())

	// Set security headers (helmet for supported, custom for others)
	app.Use(helmet.New(helmet.Config{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
	}))
	// Custom security headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com https://connect.trezor.io; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' https://connect.trezor.io wss://connect.trezor.io https://api.twitter.com;")
		return c.Next()
	})

	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(getCORSOrigins(), ","),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length, Content-Type, X-CSRF-Token",
		MaxAge:           300, // 5 minutes
		Next: func(c *fiber.Ctx) bool {
			// Skip CORS for health check endpoint
			return c.Path() == "/health"
		},
	}))

	// Debug log to confirm CSRF middleware is reached
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodPost {
			println("DEBUG: CSRF middleware reached for", c.Path(), c.Method())
		}
		return c.Next()
	})

	// Debug: print incoming CSRF cookie and header for POST requests
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodPost {
			println("[DEBUG] Incoming POST:", c.Path())
			println("[DEBUG] Cookie _csrf:", c.Cookies("_csrf"))
			println("[DEBUG] Header X-CSRF-Token:", c.Get("X-CSRF-Token"))
		}
		return c.Next()
	})

	// Custom CSRF protection that handles both form tokens and headers
	app.Use(func(c *fiber.Ctx) error {
		// For all requests, ensure we have a consistent CSRF token
		token := c.Cookies("_csrf")
		if token == "" {
			token = generateCSRFToken()
			c.Cookie(&fiber.Cookie{
				Name:     "_csrf",
				Value:    token,
				HTTPOnly: false, // JavaScript needs to read this
				Secure:   false, // Set to true in production with HTTPS
				SameSite: "Lax",
			})
		}
		c.Locals("token", token)

		// Skip CSRF validation for GET, HEAD, OPTIONS requests
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// For POST/PUT/DELETE requests, get existing token from cookie
		expectedToken := c.Cookies("_csrf")
		if expectedToken == "" {
			// No token exists, this is an error for protected requests
			println("[DEBUG] CSRF validation failed: no token in cookie")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "CSRF token missing",
			})
		}

		c.Locals("token", expectedToken)

		// Try form field first (for regular form submissions)
		formToken := c.FormValue("_token")
		if formToken != "" && formToken == expectedToken {
			println("[DEBUG] CSRF validation successful via form token")
			return c.Next()
		}

		// Try header (for HTMX/AJAX requests)
		headerToken := c.Get("X-CSRF-Token")
		if headerToken != "" && headerToken == expectedToken {
			println("[DEBUG] CSRF validation successful via header token")
			return c.Next()
		}

		// If we get here, validation failed
		println("[DEBUG] CSRF validation failed")
		println("[DEBUG] Expected token:", expectedToken)
		println("[DEBUG] Received header:", headerToken)
		println("[DEBUG] Received form:", formToken)

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "CSRF token validation failed",
		})
	})

	// Debug: print every request method and path to confirm CSRF middleware runs
	app.Use(func(c *fiber.Ctx) error {
		println("[DEBUG] CSRF middleware: method =", c.Method(), "path =", c.Path())
		return c.Next()
	})

	// Set c.Locals("csrf") for template compatibility using the same token
	app.Use(func(c *fiber.Ctx) error {
		// Use the token that was already set up in the previous middleware
		token := c.Locals("token")
		if token != nil {
			c.Locals("csrf", token)
		} else {
			c.Locals("csrf", "")
		}
		return c.Next()
	})

	// Rate limiting - configurable via environment
	maxRequests := 100 // default
	if max := os.Getenv("RATE_LIMIT_MAX"); max != "" {
		if parsed, err := strconv.ParseInt(max, 10, 32); err == nil {
			// SECURITY: Add bounds checking for integer conversion
			if parsed > 0 && parsed <= math.MaxInt32 {
				maxRequests = int(parsed)
			}
		}
	}

	expiration := 60 // default 1 minute
	if exp := os.Getenv("RATE_LIMIT_EXPIRATION"); exp != "" {
		if parsed, err := strconv.ParseInt(exp, 10, 32); err == nil {
			// SECURITY: Add bounds checking for integer conversion
			if parsed > 0 && parsed <= math.MaxInt32 {
				expiration = int(parsed)
			}
		}
	}

	app.Use(limiter.New(limiter.Config{
		Max:        maxRequests,
		Expiration: time.Duration(expiration) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP + User Agent as key for better rate limiting
			return c.IP() + ":" + c.Get("User-Agent")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
	}))
}
