package handlers

import (
	"strconv"

	"bitcoinpitch.org/internal/database"

	"github.com/gofiber/fiber/v2"
)

// TagSuggestionsHandler provides tag suggestions for autocomplete
func TagSuggestionsHandler(c *fiber.Ctx) error {
	repo := c.Locals("repo").(*database.Repository)

	query := c.Query("q", "")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if len(query) < 2 {
		return c.JSON([]interface{}{})
	}

	tags, err := repo.SearchTags(c.Context(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search tags",
		})
	}

	return c.JSON(tags)
}

// TagListHandler returns all tags with usage counts
func TagListHandler(c *fiber.Ctx) error {
	repo := c.Locals("repo").(*database.Repository)

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	tags, err := repo.ListTags(c.Context(), nil, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list tags",
		})
	}

	return c.JSON(tags)
}
