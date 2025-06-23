package handlers

import (
	"bytes"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v2"
)

// TestHandler renders the test page using Jet templates
func TestHandler(c *fiber.Ctx) error {
	// Get the Jet view from the context
	view := c.Locals("view").(*jet.Set)

	// Create template variables
	vars := make(jet.VarMap)
	vars.Set("Title", "Test Page")
	vars.Set("Description", "A test page to verify Jet template rendering")

	if csrfToken := c.Locals("csrf"); csrfToken != nil {
		vars.Set("CsrfToken", csrfToken)
	}

	// Add a flash message for testing
	vars.Set("Flash", struct {
		Type    string
		Message string
	}{
		Type:    "success",
		Message: "Template rendering test successful!",
	})

	// Render the template
	c.Type("html")
	t, err := view.GetTemplate("pages/test.jet")
	if err != nil {
		return c.SendString("TEMPLATE ERROR: " + err.Error())
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars, nil); err != nil {
		return c.SendString("TEMPLATE EXECUTION ERROR: " + err.Error())
	}
	return c.SendString(buf.String())
}
