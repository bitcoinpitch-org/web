package middleware

import (
	"context"
	"time"

	"bitcoinpitch.org/internal/database"
	"github.com/gofiber/fiber/v2"
)

// HealthStatus represents the health check response
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Services  struct {
		Database struct {
			Status string                 `json:"status"`
			Error  string                 `json:"error,omitempty"`
			Stats  map[string]interface{} `json:"stats,omitempty"`
		} `json:"database"`
	} `json:"services"`
}

// HealthCheck handles the health check endpoint
func HealthCheck(c *fiber.Ctx) error {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0", // TODO: Get from build info
	}

	// Get database from context
	db, ok := c.Locals("db").(*database.DB)
	if !ok || db == nil {
		status.Status = "degraded"
		status.Services.Database.Status = "unavailable"
		status.Services.Database.Error = "database connection not initialized"
		return c.JSON(status)
	}

	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	// Use the DB.HealthCheck method which properly manages connections
	if err := db.HealthCheck(ctx); err != nil {
		status.Status = "degraded"
		status.Services.Database.Status = "unhealthy"
		status.Services.Database.Error = err.Error()
	} else {
		status.Services.Database.Status = "healthy"
		// Add database stats
		stats := db.Stats()
		status.Services.Database.Stats = map[string]interface{}{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
		}
	}

	return c.JSON(status)
}
