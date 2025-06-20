package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"bitcoinpitch.org/internal/config"
	"bitcoinpitch.org/internal/database"
	"bitcoinpitch.org/internal/models"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AdminHandler handles admin panel operations
type AdminHandler struct {
	configService *config.Service
	repo          *database.Repository
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(configService *config.Service, repo *database.Repository) *AdminHandler {
	return &AdminHandler{
		configService: configService,
		repo:          repo,
	}
}

// AdminDashboardHandler shows the main admin dashboard
func (h *AdminHandler) AdminDashboardHandler(c *fiber.Ctx) error {
	log.Println("[DEBUG] AdminDashboardHandler called")
	view := c.Locals("view").(*jet.Set)
	user := c.Locals("user").(*models.User)
	log.Printf("[DEBUG] AdminDashboard: user=%v, role=%v", user.GetDisplayName(), user.Role)

	// Get some basic stats
	ctx := c.Context()

	// Count users by role
	adminCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleAdmin)
	modCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleModerator)
	userCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleUser)

	// Get recent config changes
	recentLogs, _ := h.repo.GetAllConfigAuditLogs(ctx, 10, 0)

	vars := make(jet.VarMap)
	vars.Set("Title", "Admin Dashboard")
	vars.Set("User", user)
	vars.Set("ShowUserMenu", true)

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en")
	}

	vars.Set("AdminCount", adminCount)
	vars.Set("ModeratorCount", modCount)
	vars.Set("UserCount", userCount)
	vars.Set("RecentConfigLogs", recentLogs)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	t, err := view.GetTemplate("pages/admin/dashboard.jet")
	if err != nil {
		log.Printf("[DEBUG] Template error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf strings.Builder
	if err := t.Execute(&buf, vars, nil); err != nil {
		log.Printf("[DEBUG] Template execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	log.Println("[DEBUG] AdminDashboard rendered successfully")
	return c.Type("html").SendString(buf.String())
}

// AdminConfigHandler shows the configuration management page
func (h *AdminHandler) AdminConfigHandler(c *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DEBUG] AdminConfigHandler PANIC: %v", r)
		}
	}()

	log.Println("[DEBUG] AdminConfigHandler called")
	view := c.Locals("view").(*jet.Set)
	user := c.Locals("user").(*models.User)

	ctx := c.Context()
	category := c.Query("category", "pitch_limits")
	log.Printf("[DEBUG] AdminConfig: category=%s", category)

	// Get settings by category
	settings, err := h.configService.GetSettingsByCategory(ctx, category)
	if err != nil {
		log.Printf("[DEBUG] AdminConfig: GetSettingsByCategory error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load configuration: " + err.Error())
	}
	log.Printf("[DEBUG] AdminConfig: found %d settings", len(settings))

	vars := make(jet.VarMap)
	vars.Set("Title", "Configuration Management")
	vars.Set("User", user)
	vars.Set("ShowUserMenu", true)

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en")
	}

	categories := models.GetConfigCategories()
	log.Printf("[DEBUG] AdminConfig: Categories type: %T, length: %d", categories, len(categories))
	for i, cat := range categories {
		log.Printf("[DEBUG] AdminConfig: Category[%d]: Name=%s, DisplayName=%s", i, cat.Name, cat.DisplayName)
	}

	vars.Set("Settings", settings)
	vars.Set("CurrentCategory", category)
	vars.Set("Categories", categories)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	t, err := view.GetTemplate("pages/admin/config.jet")
	if err != nil {
		log.Printf("[DEBUG] AdminConfig: Template error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf strings.Builder
	if err := t.Execute(&buf, vars, nil); err != nil {
		log.Printf("[DEBUG] AdminConfig: Template execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	log.Println("[DEBUG] AdminConfig rendered successfully")
	return c.Type("html").SendString(buf.String())
}

// AdminConfigUpdateHandler handles configuration updates
func (h *AdminHandler) AdminConfigUpdateHandler(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	ctx := c.Context()

	log.Printf("[DEBUG] AdminConfigUpdateHandler: Starting config update")
	log.Printf("[DEBUG] AdminConfigUpdateHandler: User: %s", user.GetDisplayName())

	// Get form values using c.FormValue for regular form data
	category := c.FormValue("category")
	if category == "" {
		log.Printf("[DEBUG] AdminConfigUpdateHandler: No category specified")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category is required",
		})
	}

	log.Printf("[DEBUG] AdminConfigUpdateHandler: Category: %s", category)

	// Get all form values and process config updates
	var updates []models.ConfigSetting

	// Parse all form values
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)

		log.Printf("[DEBUG] AdminConfigUpdateHandler: Form field: %s = %s", keyStr, valueStr)

		// Skip non-config fields
		if keyStr == "_token" || keyStr == "category" {
			return
		}

		// Extract config key from form field name (remove "config_" prefix)
		if len(keyStr) > 7 && keyStr[:7] == "config_" {
			configKey := keyStr[7:]
			log.Printf("[DEBUG] AdminConfigUpdateHandler: Processing config: %s = %s", configKey, valueStr)

			updates = append(updates, models.ConfigSetting{
				Key:   configKey,
				Value: valueStr,
			})
		}
	})

	if len(updates) == 0 {
		log.Printf("[DEBUG] AdminConfigUpdateHandler: No configuration updates found")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No configuration updates provided",
		})
	}

	log.Printf("[DEBUG] AdminConfigUpdateHandler: Processing %d config updates", len(updates))

	// Update configurations
	for _, update := range updates {
		log.Printf("[DEBUG] AdminConfigUpdateHandler: Updating %s = %s", update.Key, update.Value)

		err := h.configService.SetString(ctx, update.Key, update.Value, user.BaseModel.ID)
		if err != nil {
			log.Printf("[DEBUG] AdminConfigUpdateHandler: Failed to update %s: %v", update.Key, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to update %s: %v", update.Key, err),
			})
		}

		log.Printf("[DEBUG] AdminConfigUpdateHandler: Successfully updated %s", update.Key)
	}

	log.Printf("[DEBUG] AdminConfigUpdateHandler: All updates completed successfully")

	// Redirect back to the config page
	return c.Redirect(fmt.Sprintf("/admin/config?category=%s", category))
}

// AdminUsersHandler shows the user management page
func (h *AdminHandler) AdminUsersHandler(c *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DEBUG] AdminUsersHandler PANIC: %v", r)
		}
	}()

	log.Println("[DEBUG] AdminUsersHandler called")
	view := c.Locals("view").(*jet.Set)
	user := c.Locals("user").(*models.User)

	ctx := c.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	// Parse role filter
	roleFilter := c.Query("role", "")
	log.Printf("[DEBUG] AdminUsers: page=%d, roleFilter=%s", page, roleFilter)

	var users []*models.User
	var totalUsers int
	var err error

	// Get user statistics for the template
	adminCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleAdmin)
	modCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleModerator)
	userCount, _ := h.repo.CountUsersByRole(ctx, models.UserRoleUser)

	if roleFilter != "" {
		log.Printf("[DEBUG] AdminUsers: Getting users by role: %s", roleFilter)
		users, err = h.repo.GetUsersByRole(ctx, models.UserRole(roleFilter), limit, offset)
		if err != nil {
			log.Printf("[DEBUG] AdminUsers: GetUsersByRole error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to load users: " + err.Error())
		}
		totalUsers, err = h.repo.CountUsersByRole(ctx, models.UserRole(roleFilter))
	} else {
		log.Printf("[DEBUG] AdminUsers: Getting all users")
		users, err = h.repo.GetAllUsers(ctx, limit, offset)
		if err != nil {
			log.Printf("[DEBUG] AdminUsers: GetAllUsers error: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to load users: " + err.Error())
		}
		totalUsers, err = h.repo.CountAllUsers(ctx)
	}

	if err != nil {
		log.Printf("[DEBUG] AdminUsers: Count error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to count users: " + err.Error())
	}

	log.Printf("[DEBUG] AdminUsers: Found %d users, total: %d", len(users), totalUsers)

	totalPages := (totalUsers + limit - 1) / limit

	vars := make(jet.VarMap)
	vars.Set("Title", "User Management")
	vars.Set("User", user)
	vars.Set("CurrentUser", user)
	vars.Set("ShowUserMenu", true)

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en")
	}

	vars.Set("Users", users)
	vars.Set("TotalUsers", totalUsers)
	vars.Set("AdminCount", adminCount)
	vars.Set("ModeratorCount", modCount)
	vars.Set("UserCount", userCount)
	vars.Set("CurrentPage", page)
	vars.Set("TotalPages", totalPages)
	vars.Set("RoleFilter", roleFilter)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	log.Printf("[DEBUG] AdminUsers: About to render template")
	t, err := view.GetTemplate("pages/admin/users.jet")
	if err != nil {
		log.Printf("[DEBUG] AdminUsers: Template error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf strings.Builder
	if err := t.Execute(&buf, vars, nil); err != nil {
		log.Printf("[DEBUG] AdminUsers: Template execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	log.Printf("[DEBUG] AdminUsers: Template rendered successfully")
	return c.Type("html").SendString(buf.String())
}

// AdminUserUpdateRoleHandler handles user role updates
func (h *AdminHandler) AdminUserUpdateRoleHandler(c *fiber.Ctx) error {
	userID := c.Params("id")
	newRole := c.FormValue("role")
	currentUser := c.Locals("user").(*models.User)

	// Validate user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Validate role
	var role models.UserRole
	switch newRole {
	case "user":
		role = models.UserRoleUser
	case "moderator":
		role = models.UserRoleModerator
	case "admin":
		role = models.UserRoleAdmin
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role",
		})
	}

	// Get the target user
	targetUser, err := h.repo.GetUserByID(c.Context(), userUUID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Prevent users from changing their own role
	if targetUser.ID == currentUser.ID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot change your own role",
		})
	}

	// Update the user's role
	targetUser.SetRole(role)
	if err := h.repo.UpdateUser(c.Context(), targetUser); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user role",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("User role updated to %s", newRole),
	})
}

// AdminAuditLogsHandler shows the audit logs page
func (h *AdminHandler) AdminAuditLogsHandler(c *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DEBUG] AdminAuditLogsHandler PANIC: %v", r)
		}
	}()

	log.Println("[DEBUG] AdminAuditLogsHandler called")
	view := c.Locals("view").(*jet.Set)
	user := c.Locals("user").(*models.User)

	ctx := c.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit
	log.Printf("[DEBUG] AdminAuditLogs: page=%d, limit=%d, offset=%d", page, limit, offset)

	// Get audit logs
	logs, err := h.repo.GetAllConfigAuditLogs(ctx, limit, offset)
	if err != nil {
		log.Printf("[DEBUG] AdminAuditLogs: GetAllConfigAuditLogs error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load audit logs: " + err.Error())
	}

	log.Printf("[DEBUG] AdminAuditLogs: Found %d audit logs", len(logs))

	vars := make(jet.VarMap)
	vars.Set("Title", "Audit Logs")
	vars.Set("User", user)
	vars.Set("CurrentUser", user)
	vars.Set("ShowUserMenu", true)

	// Set current language from i18n middleware
	if currentLang := c.Locals("currentLang"); currentLang != nil {
		vars.Set("currentLang", currentLang)
	} else {
		vars.Set("currentLang", "en")
	}

	vars.Set("AuditLogs", logs)
	vars.Set("CurrentPage", page)

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	log.Printf("[DEBUG] AdminAuditLogs: About to render template")
	t, err := view.GetTemplate("pages/admin/audit-logs.jet")
	if err != nil {
		log.Printf("[DEBUG] AdminAuditLogs: Template error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template error: " + err.Error())
	}

	var buf strings.Builder
	if err := t.Execute(&buf, vars, nil); err != nil {
		log.Printf("[DEBUG] AdminAuditLogs: Template execution error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Template execution error: " + err.Error())
	}

	log.Printf("[DEBUG] AdminAuditLogs: Template rendered successfully")
	return c.Type("html").SendString(buf.String())
}
