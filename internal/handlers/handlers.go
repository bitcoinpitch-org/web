package handlers

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/middleware"
	"bitcoinpitch.org/internal/models"
	"bitcoinpitch.org/internal/validation"
)

// HomeHandler renders the home page
func HomeHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	repo := c.Locals("repo").(*database.Repository)

	// Get user from context if authenticated
	user, _ := c.Locals("user").(*models.User)

	// Create template variables
	vars := make(jet.VarMap)
	vars.Set("Title", "BitcoinPitch.org - Share Your Bitcoin Pitches")
	vars.Set("Description", "Discover and share the best Bitcoin, Lightning, and Cashu pitches. Vote on your favorites and help build the best collection of Bitcoin advocacy.")
	vars.Set("Category", "") // Set empty category for homepage so navigation works

	// Set user in template context
	if user != nil {
		println("[DEBUG] HomeHandler: User authenticated:", user.GetDisplayName())
		vars.Set("User", user)
		vars.Set("UserDisplayName", user.GetDisplayName())
		vars.Set("AuthStatus", "authenticated")
		vars.Set("ShowUserMenu", true)
	} else {
		println("[DEBUG] HomeHandler: No user found")
		vars.Set("AuthStatus", "anonymous")
		vars.Set("ShowUserMenu", false)
	}

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}

	vars.Set("DbConnected", true)
	vars.Set("Page", "home")

	// Get available languages for filter
	availableLanguages, langErr := repo.GetAvailableLanguages(c.Context())
	if langErr != nil {
		println("[DEBUG] Error getting available languages:", langErr.Error())
		availableLanguages = []string{} // fallback to empty list
	}
	vars.Set("AvailableLanguages", availableLanguages)

	// Get pagination configuration
	configService := c.Locals("configService").(*config.Service)
	paginationConfig := configService.PaginationConfig(c.Context())

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	// Determine default page size based on user preference or system default
	defaultPageSize := paginationConfig.DefaultPageSize
	if user != nil && user.GetPageSize() > 0 {
		defaultPageSize = user.GetPageSize()
	}

	pageSize, _ := strconv.Atoi(c.Query("size", strconv.Itoa(defaultPageSize)))
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > paginationConfig.MaxPageSize {
		pageSize = paginationConfig.MaxPageSize
	}

	offset := (page - 1) * pageSize

	// Get all filters from query parameters
	tagFilter := c.Query("tag")
	lengthFilter := c.Query("length")
	authorFilter := c.Query("author")
	languageFilter := c.Query("language")

	// Set filter values in template context
	vars.Set("TagFilter", tagFilter)
	vars.Set("LengthFilter", lengthFilter)
	vars.Set("AuthorFilter", authorFilter)
	vars.Set("LanguageFilter", languageFilter)

	// Build additional filters (excluding main_category)
	additionalFilters := make(map[string]interface{})

	if lengthFilter != "" {
		additionalFilters["length_category"] = lengthFilter
	}
	if languageFilter != "" {
		additionalFilters["language"] = languageFilter
	}
	if authorFilter == "me" && user != nil {
		additionalFilters["user_id"] = user.ID
	}

	// Fetch pitches for each category with pagination and filters
	var bitcoinPitches, lightningPitches, cashuPitches []models.Pitch

	// Bitcoin pitches
	bitcoinFilters := map[string]interface{}{"main_category": "bitcoin"}
	for key, value := range additionalFilters {
		bitcoinFilters[key] = value
	}

	var bitcoinPitchList []*models.Pitch
	var err error

	if tagFilter != "" {
		bitcoinPitchList, err = repo.ListPitchesByTagAndFilters(c.Context(), "bitcoin", tagFilter, additionalFilters, pageSize, offset)
	} else {
		bitcoinPitchList, err = repo.ListPitches(c.Context(), bitcoinFilters, pageSize, offset)
	}

	if err != nil {
		println("[HomeHandler] Bitcoin pitches error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch bitcoin pitches: " + err.Error())
	}

	for _, p := range bitcoinPitchList {
		p.CurrentUser = user
		if user != nil {
			currentVote, err := repo.GetVote(c.Context(), p.ID, user.ID)
			if err != nil && err != database.ErrNotFound {
				println("[DEBUG] Error getting current vote for pitch", p.ID.String(), ":", err.Error())
			}
			p.CurrentUserVote = currentVote
		}
		bitcoinPitches = append(bitcoinPitches, *p)
	}

	// Lightning pitches
	lightningFilters := map[string]interface{}{"main_category": "lightning"}
	for key, value := range additionalFilters {
		lightningFilters[key] = value
	}

	var lightningPitchList []*models.Pitch

	if tagFilter != "" {
		lightningPitchList, err = repo.ListPitchesByTagAndFilters(c.Context(), "lightning", tagFilter, additionalFilters, pageSize, offset)
	} else {
		lightningPitchList, err = repo.ListPitches(c.Context(), lightningFilters, pageSize, offset)
	}

	if err != nil {
		println("[HomeHandler] Lightning pitches error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch lightning pitches: " + err.Error())
	}

	for _, p := range lightningPitchList {
		p.CurrentUser = user
		if user != nil {
			currentVote, err := repo.GetVote(c.Context(), p.ID, user.ID)
			if err != nil && err != database.ErrNotFound {
				println("[DEBUG] Error getting current vote for pitch", p.ID.String(), ":", err.Error())
			}
			p.CurrentUserVote = currentVote
		}
		lightningPitches = append(lightningPitches, *p)
	}

	// Cashu pitches
	cashuFilters := map[string]interface{}{"main_category": "cashu"}
	for key, value := range additionalFilters {
		cashuFilters[key] = value
	}

	var cashuPitchList []*models.Pitch

	if tagFilter != "" {
		cashuPitchList, err = repo.ListPitchesByTagAndFilters(c.Context(), "cashu", tagFilter, additionalFilters, pageSize, offset)
	} else {
		cashuPitchList, err = repo.ListPitches(c.Context(), cashuFilters, pageSize, offset)
	}

	if err != nil {
		println("[HomeHandler] Cashu pitches error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch cashu pitches: " + err.Error())
	}

	for _, p := range cashuPitchList {
		p.CurrentUser = user
		if user != nil {
			currentVote, err := repo.GetVote(c.Context(), p.ID, user.ID)
			if err != nil && err != database.ErrNotFound {
				println("[DEBUG] Error getting current vote for pitch", p.ID.String(), ":", err.Error())
			}
			p.CurrentUserVote = currentVote
		}
		cashuPitches = append(cashuPitches, *p)
	}

	// Calculate total counts for pagination across all categories
	var totalBitcoinPitches, totalLightningPitches, totalCashuPitches int
	var totalPitches int

	if tagFilter != "" {
		// Count by tag and filters for each category
		totalBitcoinPitches, err = repo.CountPitchesByTagAndFilters(c.Context(), "bitcoin", tagFilter, additionalFilters)
		if err != nil {
			totalBitcoinPitches = len(bitcoinPitches)
		}
		totalLightningPitches, err = repo.CountPitchesByTagAndFilters(c.Context(), "lightning", tagFilter, additionalFilters)
		if err != nil {
			totalLightningPitches = len(lightningPitches)
		}
		totalCashuPitches, err = repo.CountPitchesByTagAndFilters(c.Context(), "cashu", tagFilter, additionalFilters)
		if err != nil {
			totalCashuPitches = len(cashuPitches)
		}
	} else {
		// Count by filters for each category
		totalBitcoinPitches, err = repo.CountPitches(c.Context(), bitcoinFilters)
		if err != nil {
			totalBitcoinPitches = len(bitcoinPitches)
		}
		totalLightningPitches, err = repo.CountPitches(c.Context(), lightningFilters)
		if err != nil {
			totalLightningPitches = len(lightningPitches)
		}
		totalCashuPitches, err = repo.CountPitches(c.Context(), cashuFilters)
		if err != nil {
			totalCashuPitches = len(cashuPitches)
		}
	}

	// Calculate total across all categories for the displayed view
	totalPitches = totalBitcoinPitches + totalLightningPitches + totalCashuPitches

	totalPages := (totalPitches + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	vars.Set("bitcoinPitches", bitcoinPitches)
	vars.Set("lightningPitches", lightningPitches)
	vars.Set("cashuPitches", cashuPitches)

	// Set pagination variables
	vars.Set("CurrentPage", page)
	vars.Set("TotalPages", totalPages)
	vars.Set("TotalPitches", totalPitches)
	vars.Set("PageSize", pageSize)
	vars.Set("PaginationConfig", paginationConfig)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		println("[DEBUG] HomeHandler: c.Locals(\"csrf\") =", csrfToken)
		vars.Set("CsrfToken", csrfToken)
	} else {
		println("[DEBUG] HomeHandler: c.Locals(\"csrf\") is nil")
	}

	t, err := view.GetTemplate("pages/home.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// PitchListHandler renders the list of pitches for a category
func PitchListHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	category := c.Path()[1:] // Remove leading slash

	// Get user from context if authenticated
	user, _ := c.Locals("user").(*models.User)

	vars := make(jet.VarMap)
	vars.Set("Title", category+" Pitches")
	vars.Set("Description", "Browse "+category+" pitches")
	vars.Set("Category", category)
	vars.Set("TagFilter", "")      // Default empty value
	vars.Set("LengthFilter", "")   // Default empty value
	vars.Set("AuthorFilter", "")   // Default empty value
	vars.Set("LanguageFilter", "") // Default empty value

	// Set user in template context
	if user != nil {
		vars.Set("User", user)
		vars.Set("UserDisplayName", user.GetDisplayName())
		vars.Set("AuthStatus", "authenticated")
		vars.Set("ShowUserMenu", true)
	} else {
		vars.Set("AuthStatus", "anonymous")
		vars.Set("ShowUserMenu", false)
	}

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en") // fallback to English
	}

	repo := c.Locals("repo").(*database.Repository)
	configService := c.Locals("configService").(*config.Service)

	// Get pagination configuration
	paginationConfig := configService.PaginationConfig(c.Context())

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	// Determine default page size based on user preference or system default
	defaultPageSize := paginationConfig.DefaultPageSize
	if user != nil && user.GetPageSize() > 0 {
		defaultPageSize = user.GetPageSize()
	}

	pageSize, _ := strconv.Atoi(c.Query("size", strconv.Itoa(defaultPageSize)))
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > paginationConfig.MaxPageSize {
		pageSize = paginationConfig.MaxPageSize
	}

	offset := (page - 1) * pageSize

	// Get all filters from query parameters
	tagFilter := c.Query("tag")
	lengthFilter := c.Query("length")
	authorFilter := c.Query("author")
	languageFilter := c.Query("language")

	// Set filter values in template context
	if tagFilter != "" {
		vars.Set("TagFilter", tagFilter)
	}
	if lengthFilter != "" {
		vars.Set("LengthFilter", lengthFilter)
	}
	if authorFilter != "" {
		vars.Set("AuthorFilter", authorFilter)
	}
	if languageFilter != "" {
		vars.Set("LanguageFilter", languageFilter)
	}

	// Get available languages for this category
	availableLanguages, err := repo.GetAvailableLanguagesByCategory(c.Context(), category)
	if err != nil {
		println("[DEBUG] Error getting available languages:", err.Error())
		availableLanguages = []string{} // fallback to empty list
	}
	vars.Set("AvailableLanguages", availableLanguages)

	// Build additional filters (excluding tag and main_category)
	additionalFilters := make(map[string]interface{})

	if lengthFilter != "" {
		additionalFilters["length_category"] = lengthFilter
	}
	if languageFilter != "" {
		additionalFilters["language"] = languageFilter
	}
	if authorFilter == "me" && user != nil {
		additionalFilters["user_id"] = user.ID
	}

	var pitches []*models.Pitch
	var totalPitches int

	if tagFilter != "" {
		// Use combined tag and filters query
		pitches, err = repo.ListPitchesByTagAndFilters(c.Context(), category, tagFilter, additionalFilters, pageSize, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch pitches: " + err.Error())
		}
		totalPitches, err = repo.CountPitchesByTagAndFilters(c.Context(), category, tagFilter, additionalFilters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to count pitches: " + err.Error())
		}
	} else {
		// Use regular filtering with main_category and all other filters
		filters := map[string]interface{}{"main_category": category}

		// Add all additional filters to the main filters map
		for key, value := range additionalFilters {
			filters[key] = value
		}

		pitches, err = repo.ListPitches(c.Context(), filters, pageSize, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch pitches: " + err.Error())
		}
		totalPitches, err = repo.CountPitches(c.Context(), filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to count pitches: " + err.Error())
		}
	}

	// Set current user for each pitch to enable edit/delete buttons
	for i := range pitches {
		pitches[i].CurrentUser = user
	}

	// Calculate pagination info
	totalPages := (totalPitches + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	// Set pagination variables
	vars.Set("pitches", pitches)
	vars.Set("CurrentPage", page)
	vars.Set("TotalPages", totalPages)
	vars.Set("TotalPitches", totalPitches)
	vars.Set("PageSize", pageSize)
	vars.Set("PaginationConfig", paginationConfig)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	// Check if this is an HTMX request for pagination
	isHTMX := c.Get("HX-Request") == "true"
	var templateName string
	if isHTMX {
		// For HTMX requests, return only the pitch list fragment
		templateName = "partials/pitch-list-fragment.jet"
	} else {
		// For regular requests, return the full page
		templateName = "pages/pitch-list.jet"
	}

	t, err := view.GetTemplate(templateName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// PitchViewHandler renders a single pitch
func PitchViewHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	pitchID := c.Params("id")

	vars := make(jet.VarMap)
	vars.Set("Title", "View Pitch")
	vars.Set("PitchID", pitchID)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	t, err := view.GetTemplate("pages/pitch-view.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// PitchEditHandler renders the pitch edit form (GET for modal, POST for update via HTMX)
func PitchEditHandler(c *fiber.Ctx) error {
	// Get repository from context
	repo := c.Locals("repo").(*database.Repository)

	// Get config service from context
	configService := c.Locals("configService").(*config.Service)

	// GET request - show the edit form
	if c.Method() == "GET" {
		// Parse pitch ID
		pitchID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid pitch ID")
		}

		// Fetch the pitch
		pitch, err := repo.GetPitch(c.Context(), pitchID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Pitch not found")
		}

		// Check ownership
		user, ok := c.Locals("user").(*models.User)
		if !ok || pitch.UserID != user.ID {
			return c.Status(fiber.StatusForbidden).SendString("Not authorized to edit this pitch")
		}

		// Render edit form
		view := c.Locals("view").(*jet.Set)
		tmpl, err := view.GetTemplate("partials/pitch-form.jet")
		if err != nil {
			log.Printf("[ERROR] PitchEditHandler: Template error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
		}

		// Create template variables
		vars := make(jet.VarMap)
		vars.Set("Title", "Edit Your Pitch")
		vars.Set("FormAction", fmt.Sprintf("/pitch/%s/edit", pitch.ID.String()))
		vars.Set("SubmitLabel", "Update Pitch")
		vars.Set("Pitch", pitch)
		vars.Set("CurrentUser", user) // Add current user for template
		vars.Set("MainCategory", pitch.MainCategory)
		vars.Set("PitchLimits", configService.PitchLimits(c.Context())) // Pass current pitch limits to template

		// Pass CSRF token to template
		if csrfToken := c.Locals("csrf"); csrfToken != nil {
			vars.Set("CsrfToken", csrfToken)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars, nil); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
		}

		return c.Type("html").Send(buf.Bytes())
	}

	// POST request - handle form submission
	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid pitch ID",
		})
	}

	// Fetch the pitch
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pitch not found",
		})
	}

	// Check ownership
	user, ok := c.Locals("user").(*models.User)
	if !ok || pitch.UserID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to edit this pitch",
		})
	}

	// Validate CSRF token
	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		token := c.FormValue("_csrf")
		if token == "" {
			token = c.Get("X-CSRF-Token")
		}
		println("[DEBUG] PitchEditHandler: expected CSRF token:", csrfToken.(string))
		println("[DEBUG] PitchEditHandler: incoming CSRF token:", token)
		if token != csrfToken.(string) {
			println("[DEBUG] PitchEditHandler: CSRF token mismatch, returning 403")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Invalid CSRF token",
			})
		}
	}

	// Parse form data
	println("[DEBUG] Raw request body:", string(c.Body()))

	// Parse form values manually to avoid tag corruption
	var input validation.PitchInput
	input.Content = c.FormValue("content")

	// ANTISPAM CHECK: Check if user can edit this pitch
	if err := middleware.CheckPitchEditLimit(c, user.ID, pitchID, input.Content); err != nil {
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

	println("[DEBUG] Parsed input struct:", fmt.Sprintf("%+v", input))

	// Calculate length category based on content length using configurable limits
	input.LengthCategory = CalculateLengthCategory(input.Content, configService)

	// Validate input using configurable limits
	if err := validation.ValidatePitchInput(input, configService); err != nil {
		println("[DEBUG] Validation error:", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Update pitch
	pitch.Content = input.Content
	pitch.Language = input.Language
	pitch.MainCategory = input.MainCategory
	pitch.LengthCategory = input.LengthCategory
	pitch.SetAuthor(input.AuthorType, input.AuthorName, input.AuthorHandle)
	pitch.Edit(input.Content)

	if len(input.Tags) > 0 {
		pitch.Tags = make([]models.Tag, len(input.Tags))
		for i, tagName := range input.Tags {
			pitch.Tags[i] = *models.NewTag(tagName)
		}
	}

	if err := repo.UpdatePitch(c.Context(), pitch); err != nil {
		println("[DEBUG] repo.UpdatePitch error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update pitch: " + err.Error(),
		})
	}

	// If HTMX request, return updated pitch card HTML
	if c.Get("HX-Request") == "true" {
		// Fetch the full pitch from DB to ensure all fields are populated
		fullPitch, err := repo.GetPitch(c.Context(), pitch.ID)
		if err != nil {
			println("[DEBUG] repo.GetPitch error after update:", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch updated pitch: " + err.Error())
		}
		println("[DEBUG] fullPitch value before rendering:", fmt.Sprintf("%+v", fullPitch))
		view := c.Locals("view").(*jet.Set)
		tmpl, err := view.GetTemplate("partials/pitch-card.jet")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
		}
		var buf bytes.Buffer
		// Pass fullPitch as root context (not as Pitch in VarMap)
		if err := tmpl.Execute(&buf, nil, fullPitch); err != nil {
			println("[DEBUG] tmpl.Execute error:", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
		}
		return c.Type("html").Send(buf.Bytes())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Pitch updated successfully",
	})
}

// PitchDeleteConfirmHandler renders the delete confirmation modal
func PitchDeleteConfirmHandler(c *fiber.Ctx) error {
	// Use idOrAny to match any string, not just valid UUIDs (see routes.go)
	idParam := c.Params("idOrAny")
	if idParam == "" {
		// Fallback: also handle /pitch/delete-confirm (no ID provided)
		idParam = c.Params("id")
	}
	println("[DEBUG] PitchDeleteConfirmHandler called for:", c.OriginalURL(), "HX-Request:", c.Get("HX-Request"), "idOrAny:", idParam)
	view := c.Locals("view").(*jet.Set)
	repo := c.Locals("repo").(*database.Repository)
	isHTMX := c.Get("HX-Request") == "true"
	if idParam == "" {
		if isHTMX {
			return c.Status(200).Type("html").SendString(`<div class='modal-header'><h2>Error</h2></div><div class='modal-body'><p>Missing pitch ID.</p></div><div class='form-actions'><button type='button' class='button secondary close-modal'>Close</button></div>`)
		}
		return c.Status(fiber.StatusBadRequest).SendString("Missing pitch ID")
	}
	pitchID, err := uuid.Parse(idParam)
	if err != nil {
		if isHTMX {
			return c.Status(200).Type("html").SendString(`<div class='modal-header'><h2>Error</h2></div><div class='modal-body'><p>Invalid pitch ID.</p></div><div class='form-actions'><button type='button' class='button secondary close-modal'>Close</button></div>`)
		}
		return c.Status(fiber.StatusBadRequest).SendString("Invalid pitch ID")
	}
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		if isHTMX {
			return c.Status(200).Type("html").SendString(`<div class='modal-header'><h2>Error</h2></div><div class='modal-body'><p>Pitch not found.</p></div><div class='form-actions'><button type='button' class='button secondary close-modal'>Close</button></div>`)
		}
		return c.Status(fiber.StatusNotFound).SendString("Pitch not found")
	}
	println("[DEBUG] PitchDeleteConfirmHandler: pitch value before rendering:", fmt.Sprintf("%+v", pitch))
	vars := make(jet.VarMap)
	vars.Set("Pitch", pitch)
	vars.Set("CsrfToken", c.Locals("csrf"))
	buf := &bytes.Buffer{}
	tmpl, err := view.GetTemplate("partials/delete-confirm.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}
	if err := tmpl.Execute(buf, vars, nil); err != nil {
		println("[DEBUG] PitchDeleteConfirmHandler: tmpl.Execute error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}
	return c.Type("html").Send(buf.Bytes())
}

// PitchDeleteHandler handles the POST request to soft delete a pitch
func PitchDeleteHandler(c *fiber.Ctx) error {
	repo := c.Locals("repo").(*database.Repository)

	// Require authentication for deleting pitches
	user, userOk := c.Locals("user").(*models.User)
	if !userOk {
		return c.Status(fiber.StatusUnauthorized).SendString("Authentication required to delete a pitch. Please log in.")
	}

	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid pitch ID")
	}

	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Pitch not found")
	}

	// Check ownership
	if pitch.UserID != user.ID {
		return c.Status(fiber.StatusForbidden).SendString("Not authorized to delete this pitch")
	}

	if err := repo.DeletePitch(c.Context(), pitchID); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete pitch: " + err.Error())
	}

	// For HTMX, return empty 204 so the card is removed
	return c.SendStatus(fiber.StatusNoContent)
}

// PitchVoteHandler handles HTMX voting requests
func PitchVoteHandler(c *fiber.Ctx) error {
	println("[DEBUG] PitchVoteHandler called:", c.Method(), c.OriginalURL())
	repo := c.Locals("repo").(*database.Repository)
	view := c.Locals("view").(*jet.Set)

	// CSRF validation is handled by Fiber's CSRF middleware automatically

	// Require authentication for voting
	user, userOk := c.Locals("user").(*models.User)
	if !userOk {
		return c.Status(fiber.StatusUnauthorized).SendString("Authentication required to vote. Please log in.")
	}

	// Parse pitch ID
	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid pitch ID")
	}

	// ANTISPAM CHECK: Check if user can vote
	if err := middleware.CheckVoteLimit(c, pitchID); err != nil {
		return err // Error response already sent by middleware
	}

	// Parse vote type from JSON body (hx-vals sends JSON)
	var body struct {
		Type string `json:"type"`
	}
	if err := c.BodyParser(&body); err != nil {
		println("[DEBUG] Error parsing JSON body:", err.Error())
		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	}
	voteType := models.VoteType(body.Type)
	println("[DEBUG] PitchVoteHandler: pitchID =", pitchID.String(), "voteType =", string(voteType))
	if voteType != models.VoteTypeUp && voteType != models.VoteTypeDown {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid vote type")
	}

	// Get existing vote if any
	existingVote, err := repo.GetVote(c.Context(), pitchID, user.ID)
	if err != nil && err != database.ErrNotFound {
		println("[DEBUG] Error getting existing vote:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to check existing vote: " + err.Error())
	}

	// Handle vote logic
	if existingVote != nil {
		println("[DEBUG] Found existing vote:", string(existingVote.VoteType))
		// If same vote type, remove vote (toggle off)
		if existingVote.VoteType == voteType {
			println("[DEBUG] Removing vote (toggle off)")
			if err := repo.DeleteVote(c.Context(), existingVote); err != nil {
				println("[DEBUG] Error deleting vote:", err.Error())
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to remove vote")
			}
		} else {
			// If different vote type, update vote (change from up to down or vice versa)
			println("[DEBUG] Updating vote from", string(existingVote.VoteType), "to", string(voteType))
			existingVote.VoteType = voteType
			existingVote.UpdatedAt = time.Now()
			if err := repo.UpdateVote(c.Context(), existingVote); err != nil {
				println("[DEBUG] Error updating vote:", err.Error())
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to update vote")
			}
		}
	} else {
		// Create new vote
		println("[DEBUG] Creating new vote:", string(voteType))
		vote := models.NewVote(pitchID, user.ID, voteType)
		if err := repo.CreateVote(c.Context(), vote); err != nil {
			println("[DEBUG] Error creating vote:", err.Error())
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create vote")
		}
	}

	// Get updated pitch with new vote counts
	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		println("[DEBUG] Error getting updated pitch:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch updated pitch: " + err.Error())
	}

	// Get current user's vote state for UI
	currentVote, err := repo.GetVote(c.Context(), pitchID, user.ID)
	if err != nil && err != database.ErrNotFound {
		println("[DEBUG] Error getting current vote:", err.Error())
	}

	// Pass the full pitch with current user context for template
	pitch.CurrentUser = user
	pitch.CurrentUserVote = currentVote

	println("[DEBUG] Vote counts - Up:", pitch.UpvoteCount, "Down:", pitch.DownvoteCount, "Score:", pitch.Score)
	if currentVote != nil {
		println("[DEBUG] User has voted:", string(currentVote.VoteType))
	} else {
		println("[DEBUG] User has not voted")
	}

	// Render the vote section template
	println("[DEBUG] About to render vote-section.jet template")

	tmpl, err := view.GetTemplate("partials/vote-section.jet")
	if err != nil {
		println("[DEBUG] Template loading error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil, pitch); err != nil {
		println("[DEBUG] Template execution error:", err.Error())
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	println("[DEBUG] Template rendered successfully, size:", buf.Len(), "bytes")
	return c.Type("html").Send(buf.Bytes())
}

// PitchCatchAllHandler handles unmatched /pitch/* routes for modal/HTMX 404s
func PitchCatchAllHandler(c *fiber.Ctx) error {
	path := c.OriginalURL()
	isHTMX := c.Get("HX-Request") == "true"
	println("[DEBUG] PitchCatchAllHandler called for:", path, "HX-Request:", c.Get("HX-Request"))
	if strings.HasSuffix(path, "/delete-confirm") && isHTMX {
		return c.Status(200).Type("html").SendString(`<div class='modal-header'><h2>Error</h2></div><div class='modal-body'><p>Pitch not found or invalid request.</p></div><div class='form-actions'><button type='button' class='button secondary close-modal'>Close</button></div>`)
	}
	return c.Status(fiber.StatusNotFound).SendString("Not found")
}

// NotFoundHandler is a global 404 handler aware of HTMX/modal requests
func NotFoundHandler(c *fiber.Ctx) error {
	path := c.OriginalURL()
	isHTMX := c.Get("HX-Request") == "true"
	println("[DEBUG] NotFoundHandler called for:", path, "HX-Request:", c.Get("HX-Request"))
	if strings.HasSuffix(path, "/delete-confirm") && isHTMX {
		c.Set("HX-Trigger", "htmx-delete-modal-error")
		println("[DEBUG] NotFoundHandler: Triggering htmx-delete-modal-error event for modal error fragment")
		return c.Status(200).Type("html").SendString(`<div class='modal-header'><h2>Error</h2></div><div class='modal-body'><p>Pitch not found or invalid request.</p></div><div class='form-actions'><button type='button' class='button secondary close-modal'>Close</button></div>`)
	}
	// Render normal 404 page (Jet template or plain text)
	return c.Status(fiber.StatusNotFound).SendString("404 - Page Not Found")
}

// PitchShareHandler renders a single pitch page for sharing
func PitchShareHandler(c *fiber.Ctx) error {
	view := c.Locals("view").(*jet.Set)
	repo := c.Locals("repo").(*database.Repository)

	pitchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid pitch ID")
	}

	pitch, err := repo.GetPitch(c.Context(), pitchID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Pitch not found")
	}

	// Manual string truncation for title and description
	titleContent := pitch.Content
	if len(titleContent) > 60 {
		titleContent = titleContent[:60] + "..."
	}
	descContent := pitch.Content
	if len(descContent) > 160 {
		descContent = descContent[:160] + "..."
	}

	vars := make(jet.VarMap)
	vars.Set("Title", fmt.Sprintf("%s | BitcoinPitch.org", titleContent))
	vars.Set("Description", descContent)
	vars.Set("pitch", pitch)

	// Add request info for Open Graph meta tags
	vars.Set("request", map[string]interface{}{
		"Scheme": c.Protocol(),
		"Host":   c.Hostname(),
	})

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	t, err := view.GetTemplate("pages/pitch-detail.jet")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	return c.Type("html").Send(buf.Bytes())
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
