package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AuthMiddleware handles authentication for protected routes
func AuthMiddleware(repo *database.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get session token from cookie
		sessionToken := c.Cookies("session_token")

		if sessionToken == "" {
			return c.Next()
		}

		// Get session from database
		session, err := repo.GetSessionByToken(c.Context(), sessionToken)
		if err != nil {
			// Invalid session, clear cookie and continue as unauthenticated
			c.ClearCookie("session_token")
			return c.Next()
		}

		// Check if session is expired
		if session.ExpiresAt.Before(time.Now()) {
			// Session expired, clean up
			repo.DeleteSession(c.Context(), session.BaseModel.ID)
			c.ClearCookie("session_token")
			return c.Next()
		}

		// Get user from database
		user, err := repo.GetUserByID(c.Context(), session.UserID)
		if err != nil {
			// User not found, clean up session
			repo.DeleteSession(c.Context(), session.BaseModel.ID)
			c.ClearCookie("session_token")
			return c.Next()
		}

		// Set user in context
		c.Locals("user", user)
		return c.Next()
	}
}

// RequireAuthMiddleware ensures user is authenticated
func RequireAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := c.Locals("user").(*models.User); !ok {
			// User not authenticated, return 401
			if c.Get("HX-Request") == "true" {
				// HTMX request, return fragment
				return c.Status(fiber.StatusUnauthorized).SendString(`
					<div class="auth-error">
						You must be logged in to perform this action.
					</div>
				`)
			}
			// Regular request, redirect to login
			return c.Redirect("/auth/login", http.StatusTemporaryRedirect)
		}
		return c.Next()
	}
}

// CreateSession creates a new session for the user
func CreateSession(repo *database.Repository, ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	// Create session
	session := &models.Session{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}

	err := repo.CreateSession(ctx, session)
	if err != nil {
		return "", err
	}

	return token, nil
}

// SetSessionCookie sets the session cookie
func SetSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HTTPOnly: true,
		Secure:   strings.HasPrefix(c.BaseURL(), "https"),
		SameSite: "Lax",
	})
}
