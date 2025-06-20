package handlers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/middleware"
	"bitcoinpitch.org/internal/models"
	"bitcoinpitch.org/internal/validation"
)

// PitchFormHandler handles the GET request for the pitch form modal
func PitchFormHandler(c *fiber.Ctx) error {
	// Get the Jet view from the context
	view := c.Locals("view").(*jet.Set)

	// Get config service from context
	configService := c.Locals("configService").(*config.Service)

	// Get the template
	tmpl, err := view.GetTemplate("partials/pitch-form.jet")
	if err != nil {
		return err
	}

	// Get category from query parameter, default to bitcoin if not provided
	category := c.Query("category", "bitcoin")
	var mainCategory models.MainCategory
	switch category {
	case "lightning":
		mainCategory = models.MainCategoryLightning
	case "cashu":
		mainCategory = models.MainCategoryCashu
	default:
		mainCategory = models.MainCategoryBitcoin
	}

	// Get current pitch limits
	limits := configService.PitchLimits(c.Context())

	// Create template variables
	vars := make(jet.VarMap)
	vars.Set("Title", "Add Your Pitch")
	vars.Set("FormAction", "/pitch/add")
	vars.Set("SubmitLabel", "Submit Pitch")
	vars.Set("Pitch", models.Pitch{})
	vars.Set("MainCategory", mainCategory)
	vars.Set("Category", category)  // Also pass the category string for the template
	vars.Set("PitchLimits", limits) // Pass current pitch limits to template

	// Pass current language for i18n
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}

	// Pass CSRF token to template
	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	// Pass auth state to template
	var currentUser *models.User
	if user, ok := c.Locals("user").(*models.User); ok {
		vars.Set("IsAuthenticated", true)
		currentUser = user
	} else {
		vars.Set("IsAuthenticated", false)
	}
	// Always set CurrentUser (even if nil)
	vars.Set("CurrentUser", currentUser)

	// Render the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars, nil); err != nil {
		return err
	}

	c.Type("html")
	return c.Send(buf.Bytes())
}

// PitchAddHandler handles the POST request for adding a new pitch
func PitchAddHandler(c *fiber.Ctx) error {
	// Validate CSRF token
	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		token := c.FormValue("_csrf")
		if token == "" {
			token = c.Get("X-CSRF-Token")
		}
		if token != csrfToken.(string) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Invalid CSRF token",
			})
		}
	}

	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get config service from context
	configService := c.Locals("configService").(*config.Service)

	// Require authentication for adding pitches
	user, ok := c.Locals("user").(*models.User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required to add a pitch. Please log in.",
		})
	}

	// Parse form values manually to avoid tag corruption
	var input validation.PitchInput
	input.Content = c.FormValue("content")

	// DEBUG: Character count discrepancy investigation
	println("[DEBUG] Form content length:", len(input.Content))
	println("[DEBUG] Form content (hex dump first 20 bytes):", fmt.Sprintf("%x", []byte(input.Content[:min(20, len(input.Content))])))
	println("[DEBUG] Form content (hex dump last 20 bytes):", fmt.Sprintf("%x", []byte(input.Content[max(0, len(input.Content)-20):])))

	// Check for specific problematic characters
	for i, r := range input.Content {
		if r == '\r' || r == '\n' || r == '\t' || r < 32 {
			println(fmt.Sprintf("[DEBUG] Found control char at pos %d: %d (%c)", i, int(r), r))
		}
	}

	// Trim leading/trailing whitespace to match frontend behavior
	input.Content = strings.TrimSpace(input.Content)
	println("[DEBUG] After TrimSpace length:", len(input.Content))

	// ANTISPAM CHECK: Check if user can create a pitch
	if err := middleware.CheckPitchCreationLimit(c, input.Content); err != nil {
		return err // Error response already sent by middleware
	}

	input.Language = c.FormValue("language")
	input.MainCategory = models.MainCategory(c.FormValue("main_category"))
	input.AuthorType = models.AuthorType(c.FormValue("author_type"))

	// Parse author fields based on type
	if authorName := c.FormValue("author_name"); authorName != "" {
		input.AuthorName = &authorName
	}
	if authorHandle := c.FormValue("author_handle"); authorHandle != "" {
		input.AuthorHandle = &authorHandle
	}

	// Handle tags: split comma-separated string into array
	if tagsStr := c.FormValue("tags"); tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		var cleanTags []string
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				cleanTags = append(cleanTags, tag)
			}
		}
		input.Tags = cleanTags
	}

	// Calculate length category based on content length using configurable limits
	input.LengthCategory = CalculateLengthCategory(input.Content, configService)

	// Validate input using configurable limits
	if err := validation.ValidatePitchInput(input, configService); err != nil {
		// Log the validation error for debugging
		println("[PitchAddHandler] Validation error:", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create pitch
	pitch := models.NewPitch(
		user.ID,
		user.ID, // posted_by is same as user_id for now
		input.Content,
		input.Language,
		input.MainCategory,
		input.LengthCategory,
		input.AuthorType,
	)

	// Set author details if provided
	if input.AuthorName != nil || input.AuthorHandle != nil {
		pitch.SetAuthor(input.AuthorType, input.AuthorName, input.AuthorHandle)
	}

	// Add tags if provided
	if len(input.Tags) > 0 {
		pitch.Tags = make([]models.Tag, len(input.Tags))
		for i, tagName := range input.Tags {
			pitch.Tags[i] = *models.NewTag(tagName)
		}
	}

	println("[DEBUG] Entered PitchAddHandler for", c.Method(), c.OriginalURL())

	// Save to database
	if err := repo.CreatePitch(c.Context(), pitch); err != nil {
		println("[DEBUG] repo.CreatePitch error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create pitch: " + err.Error(),
		})
	}

	// Record content hash for future duplicate detection
	middleware.RecordContentHash(c, user.ID, input.Content, pitch.ID)

	// Check if this is an HTMX request
	if c.Get("HX-Request") == "true" {
		// Determine main category from input
		mainCategory := input.MainCategory
		if mainCategory == "" {
			mainCategory = pitch.MainCategory
		}
		// Fetch all pitches for this category
		filters := map[string]interface{}{"main_category": mainCategory}
		pitches, err := repo.ListPitches(c.Context(), filters, 100, 0)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch pitch list: " + err.Error())
		}
		view := c.Locals("view").(*jet.Set)
		// Render only the pitch cards for this tab (not the whole page)
		var buf bytes.Buffer
		for _, p := range pitches {
			tmpl, err := view.GetTemplate("partials/pitch-card.jet")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
			}
			if err := tmpl.Execute(&buf, nil, p); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
			}
		}
		c.Type("html")
		return c.Send(buf.Bytes())
	}

	// Return JSON response for API requests
	return c.Status(fiber.StatusCreated).JSON(pitch)
}
