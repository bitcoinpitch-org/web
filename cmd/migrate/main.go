package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Parse command line flags
	up := flag.Bool("up", false, "Run migrations up")
	down := flag.Bool("down", false, "Run migrations down")
	flag.Parse()

	if !*up && !*down {
		fmt.Println("Please specify either -up or -down")
		os.Exit(1)
	}

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Get migrations directory
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		log.Fatalf("Migrations directory %s does not exist", migrationsDir)
	}

	// Create migrate instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		dbURL,
	)
	if err != nil {
		log.Fatalf("Error creating migrate instance: %v", err)
	}
	defer m.Close()

	// Run migrations
	if *up {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Error running migrations up: %v", err)
		}
		fmt.Println("Migrations up completed successfully")
	} else if *down {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Error running migrations down: %v", err)
		}
		fmt.Println("Migrations down completed successfully")
	}
}
