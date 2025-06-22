package middleware

import (
	"bitcoinpitch.org/internal/i18n"
	"github.com/gofiber/fiber/v2"
)

// I18nConfig holds the configuration for the I18n middleware
type I18nConfig struct {
	I18nManager *i18n.Manager
	CookieName  string
	DefaultLang string
}

// I18n middleware handles language detection and sets the current language in context
func I18n(config I18nConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var currentLang string
		var isFirstVisit bool

		// 1. Try to get language from cookie first
		if cookieLang := c.Cookies(config.CookieName); cookieLang != "" {
			if isValidLanguage(config.I18nManager, cookieLang) {
				currentLang = cookieLang
			}
		} else {
			// No language cookie exists - this is a first visit
			isFirstVisit = true
		}

		// 2. If no valid cookie (first visit), try to detect from Accept-Language header
		if currentLang == "" {
			acceptLang := c.Get("Accept-Language")
			detectedLang := config.I18nManager.DetectLanguageFromAccept(acceptLang)

			// Only use detected language if we have translation for it
			if detectedLang != "" && isValidLanguage(config.I18nManager, detectedLang) {
				currentLang = detectedLang
			}
		}

		// 3. Fallback to default language if nothing else works
		if currentLang == "" {
			currentLang = config.DefaultLang
		}

		// 4. If this is a first visit and we detected a valid language, set the cookie
		if isFirstVisit && currentLang != "" {
			c.Cookie(&fiber.Cookie{
				Name:     config.CookieName,
				Value:    currentLang,
				MaxAge:   30 * 24 * 60 * 60, // 30 days
				HTTPOnly: false,             // Allow JavaScript to read for client-side logic
				SameSite: "Lax",
			})
		}

		// Set the current language in context for handlers and templates
		c.Locals("currentLang", currentLang)
		c.Locals("i18nManager", config.I18nManager)
		c.Locals("isFirstVisit", isFirstVisit)

		// Add translation helper function to context
		c.Locals("t", func(key string) string {
			return config.I18nManager.T(currentLang, key)
		})

		return c.Next()
	}
}

// isValidLanguage checks if a language code is supported
func isValidLanguage(manager *i18n.Manager, langCode string) bool {
	availableLanguages := manager.GetAvailableLanguages()
	for _, lang := range availableLanguages {
		if lang == langCode {
			return true
		}
	}
	return false
}

// SetLanguage is a handler to change the user's language preference
func SetLanguage(config I18nConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		langCode := c.Params("lang")

		// Validate language code
		if !isValidLanguage(config.I18nManager, langCode) {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid language code",
			})
		}

		// Set cookie for language preference (30 days)
		c.Cookie(&fiber.Cookie{
			Name:     config.CookieName,
			Value:    langCode,
			MaxAge:   30 * 24 * 60 * 60, // 30 days
			HTTPOnly: false,             // Allow JavaScript to read for client-side logic
			SameSite: "Lax",
		})

		// Get the referer URL to redirect back
		referer := c.Get("Referer")
		if referer != "" {
			return c.Redirect(referer)
		}

		// Fallback to home page
		return c.Redirect("/")
	}
}
