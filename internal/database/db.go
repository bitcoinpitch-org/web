package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func init() {
	log.Printf("Registered SQL drivers: %v", sql.Drivers())
}

// Config holds the database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DB wraps the sqlx.DB with additional methods
type DB struct {
	*sqlx.DB
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return &DB{db}, nil
}

// WithTx executes a function within a transaction
func (db *DB) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := db.DB.Beginx()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("error rolling back transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	log.Printf("DB.Close() called: closing database connection\n%s", debug.Stack())
	return db.DB.Close()
}

// Ping checks if the database connection is alive
func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("error getting connection from pool: %w", err)
	}
	defer conn.Close()

	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	return nil
}

// RunMigrations runs all database migrations automatically using a separate connection
func RunMigrations(cfg Config, migrationsPath string) error {
	// Create a separate database connection ONLY for migrations
	migrationDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	migrationDB, err := sql.Open("postgres", migrationDSN)
	if err != nil {
		return fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close() // Safe to close this separate connection

	// Create file source
	source, err := (&file.File{}).Open("file://" + migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to open migration source: %w", err)
	}
	defer source.Close()

	// Create database driver using the separate connection
	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Create migrator
	migrator, err := migrate.NewWithInstance("file", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer migrator.Close() // Safe to close since it uses separate connection

	// Run migrations
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Log success
	version, dirty, err := migrator.Version()
	if err != nil {
		log.Printf("Migration completed, but could not get version: %v", err)
	} else {
		log.Printf("Migrations completed successfully. Current version: %d, dirty: %t", version, dirty)
	}

	return nil
}
