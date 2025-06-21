package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"bitcoinpitch.org/internal/models"
)

// PitchData represents a pitch to be created
type PitchData struct {
	Content        string
	Language       string
	MainCategory   models.MainCategory
	LengthCategory models.LengthCategory
	AuthorType     models.AuthorType
	AuthorName     *string
	AuthorHandle   *string
	Tags           []string
}

// Build database connection string from environment variables
func buildDBConnStr() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "bitcoinpitch")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "bitcoinpitch")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// If running from host machine and DB_HOST is 'db' (Docker service name),
	// use localhost instead since we're connecting from outside Docker
	if host == "db" {
		host = "localhost"
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

// Helper function to get environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	// Load .env file if it exists (look in parent directory since script runs from scripts/)
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found in parent directory, using system environment variables")
	}

	// Build database connection string from environment
	dbConnStr := buildDBConnStr()
	log.Printf("Connecting to database: host=%s, port=%s, user=%s, dbname=%s",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "bitcoinpitch"),
		getEnv("DB_NAME", "bitcoinpitch"))

	// Connect to database
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Clean up any duplicate nostr users first
	err = cleanupDuplicateNostrUsers(db)
	if err != nil {
		log.Printf("Warning: Failed to cleanup duplicate users: %v", err)
	}

	// Get the nostr user ID
	nostrUserID, err := getNostrUserID(db)
	if err != nil {
		log.Fatalf("Failed to get nostr user: %v", err)
	}

	// Define initial pitches based on web search results and Bitcoin knowledge
	pitches := []PitchData{
		// BITCOIN CATEGORY - ENGLISH
		{
			Content:        "Bitcoin: Digital gold for the digital age.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "digital-gold", "money", "general", "beginners"},
		},
		{
			Content:        "Bitcoin is the first decentralized digital currency that operates without a central bank or administrator.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "decentralized", "currency", "general", "beginners"},
		},
		{
			Content:        "Bitcoin represents a fundamental shift in how we think about money. It's not just a new payment system, but a new form of money that's scarce, divisible, portable, and verifiable. Unlike fiat currencies that can be printed infinitely, Bitcoin has a fixed supply of 21 million coins.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "money", "scarcity", "investors", "business"},
		},
		{
			Content:        "Bitcoin is the internet of money. Just as the internet democratized information, Bitcoin democratizes money.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "internet", "democracy", "general", "beginners"},
		},
		{
			Content:        "Bitcoin enables financial sovereignty.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "banking", "sovereignty", "general", "beginners"},
		},
		{
			Content:        "Bitcoin fixes this.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "fixes", "meme", "general", "developers"},
		},
		{
			Content:        "Bitcoin is the solution to the Byzantine Generals' Problem in digital money.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "consensus", "byzantine", "developers", "experts"},
		},
		{
			Content:        "Bitcoin's proof-of-work creates real-world value through energy expenditure, making it the most secure monetary network ever created.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "proof-of-work", "security", "investors", "experts"},
		},

		// LIGHTNING CATEGORY - ENGLISH
		{
			Content:        "Lightning: Bitcoin's speed layer.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "scaling", "general", "beginners"},
		},
		{
			Content:        "Lightning Network enables instant, low-cost Bitcoin transactions through payment channels.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "payments", "channels", "merchants", "business"},
		},
		{
			Content:        "The Lightning Network is Bitcoin's Layer 2 scaling solution that enables micropayments and instant transactions. By opening payment channels between parties, users can transact instantly without waiting for blockchain confirmations, while maintaining Bitcoin's security guarantees.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "layer2", "developers", "investors"},
		},
		{
			Content:        "Lightning makes Bitcoin usable for everyday purchases with instant confirmations and minimal fees.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "payments", "fees", "merchants", "general"},
		},
		{
			Content:        "Lightning enables instant payments.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "payments", "general", "merchants"},
		},
		{
			Content:        "Lightning channels enable trustless, instant Bitcoin transactions between any two parties.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "channels", "trustless", "developers", "experts"},
		},

		// CASHU CATEGORY - ENGLISH
		{
			Content:        "Cashu: Privacy-first Bitcoin.",
			Language:       "en",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "privacy", "general", "beginners"},
		},
		{
			Content:        "Cashu is a privacy-focused Bitcoin protocol using Chaumian ecash for anonymous transactions.",
			Language:       "en",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "ecash", "privacy", "developers", "experts"},
		},
		{
			Content:        "Cashu brings true privacy to Bitcoin through Chaumian ecash technology. Users can mint and spend Bitcoin tokens anonymously, with no transaction history visible on the blockchain. Perfect for those who value financial privacy.",
			Language:       "en",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "privacy", "regulators", "experts"},
		},
		{
			Content:        "Cashu enables private Bitcoin transactions using Chaumian ecash technology.",
			Language:       "en",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "privacy", "ecash", "developers", "experts"},
		},
		{
			Content:        "Privacy is a human right.",
			Language:       "en",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "privacy", "rights", "general", "regulators"},
		},

		// BITCOIN CATEGORY - CZECH
		{
			Content:        "Bitcoin: Digitální zlato pro digitální věk.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "digitální-zlato", "peníze", "general", "beginners"},
		},
		{
			Content:        "Bitcoin je první decentralizovaná digitální měna, která funguje bez centrální banky.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "decentralizace", "měna", "general", "beginners"},
		},
		{
			Content:        "Bitcoin představuje zásadní změnu v tom, jak přemýšlíme o penězích. Není to jen nový platební systém, ale nová forma peněz, která je vzácná, dělitelná, přenosná a ověřitelná.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "peníze", "vzácnost", "investors", "business"},
		},
		{
			Content:        "Bitcoin je internet peněz. Stejně jako internet demokratizoval informace, Bitcoin demokratizuje peníze.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "internet", "demokracie", "general", "beginners"},
		},
		{
			Content:        "Bitcoin umožňuje finanční svrchovanost.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "bankovnictví", "svrchovanost", "general", "beginners"},
		},
		{
			Content:        "Bitcoin to opravuje.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"bitcoin", "opravuje", "meme", "general", "developers"},
		},

		// LIGHTNING CATEGORY - CZECH
		{
			Content:        "Lightning: Bitcoinova rychlostní vrstva.",
			Language:       "cs",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "škálování", "general", "beginners"},
		},
		{
			Content:        "Lightning Network umožňuje okamžité Bitcoin transakce s nízkými poplatky.",
			Language:       "cs",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "platby", "kanály", "merchants", "business"},
		},
		{
			Content:        "Lightning Network je Bitcoinova Layer 2 řešení pro škálování, které umožňuje mikroplatby a okamžité transakce.",
			Language:       "cs",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryElevator,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "bitcoin", "layer2", "developers", "investors"},
		},
		{
			Content:        "Lightning umožňuje okamžité platby.",
			Language:       "cs",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"lightning", "platby", "general", "merchants"},
		},

		// CASHU CATEGORY - CZECH
		{
			Content:        "Cashu: Bitcoin s důrazem na soukromí.",
			Language:       "cs",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "soukromí", "general", "beginners"},
		},
		{
			Content:        "Cashu je Bitcoin protokol zaměřený na soukromí pomocí Chaumian ecash.",
			Language:       "cs",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategorySMS,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "bitcoin", "ecash", "soukromí", "developers", "experts"},
		},
		{
			Content:        "Soukromí je lidské právo.",
			Language:       "cs",
			MainCategory:   models.MainCategoryCashu,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeSame,
			Tags:           []string{"cashu", "soukromí", "práva", "general", "regulators"},
		},

		// FAMOUS QUOTES - ENGLISH
		{
			Content:        "Bitcoin is a technological tour de force.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Bill Gates"),
			Tags:           []string{"bitcoin", "technology", "quote", "investors", "business"},
		},
		{
			Content:        "Bitcoin is the most important invention since the internet.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Roger Ver"),
			Tags:           []string{"bitcoin", "invention", "internet", "quote", "investors", "general"},
		},
		{
			Content:        "Bitcoin is the internet of money.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Andreas Antonopoulos"),
			Tags:           []string{"bitcoin", "internet", "money", "quote", "general", "beginners"},
		},
		{
			Content:        "Bitcoin fixes this.",
			Language:       "en",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryOneLiner,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Bitcoin Community"),
			Tags:           []string{"bitcoin", "fixes", "meme", "quote", "general", "developers"},
		},
		{
			Content:        "Lightning Network is Bitcoin's killer app.",
			Language:       "en",
			MainCategory:   models.MainCategoryLightning,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Elizabeth Stark"),
			Tags:           []string{"lightning", "bitcoin", "killer-app", "quote", "investors", "developers"},
		},

		// FAMOUS QUOTES - CZECH
		{
			Content:        "Bitcoin je technologický zázrak.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Bill Gates"),
			Tags:           []string{"bitcoin", "technologie", "citát", "investors", "business"},
		},
		{
			Content:        "Bitcoin je nejdůležitější vynález od internetu.",
			Language:       "cs",
			MainCategory:   models.MainCategoryBitcoin,
			LengthCategory: models.LengthCategoryTweet,
			AuthorType:     models.AuthorTypeCustom,
			AuthorName:     stringPtr("Roger Ver"),
			Tags:           []string{"bitcoin", "vynález", "internet", "citát", "investors", "general"},
		},
	}

	// Create pitches
	for _, pitchData := range pitches {
		err := createPitch(db, nostrUserID, pitchData)
		if err != nil {
			log.Printf("Failed to create pitch: %v", err)
			continue
		}
		fmt.Printf("Created pitch: %s\n", truncateString(pitchData.Content, 50))
	}

	fmt.Println("Initial pitches creation completed!")
}

func getNostrUserID(db *sqlx.DB) (uuid.UUID, error) {
	// Use the actual hex pubkey from the database
	hexPubkey := "3cb60326b156fad8b996dfae73bb820dbdde1f0ac23f76c85b78a9b9813b7b3e"

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE auth_type = 'nostr' AND auth_id = $1`
	err := db.Get(&userID, query, hexPubkey)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, fmt.Errorf("nostr user not found with auth_id: %s", hexPubkey)
		}
		return uuid.Nil, err
	}
	return userID, nil
}

func createNostrUser(db *sqlx.DB, hexPubkey, npub string) (uuid.UUID, error) {
	userID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO users (id, auth_type, auth_id, username, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.Exec(query,
		userID,
		"nostr",
		hexPubkey, // Use hex format for auth_id (consistent with auth system)
		npub,      // Use npub for username
		npub,      // Use npub for display_name
		now,
		now,
	)

	if err != nil {
		return uuid.Nil, err
	}

	fmt.Printf("Created nostr user: %s (hex: %s, npub: %s)\n", userID, hexPubkey, npub)
	return userID, nil
}

func createPitch(db *sqlx.DB, userID uuid.UUID, pitchData PitchData) error {
	ctx := context.Background()

	// Start transaction
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create pitch
	pitchID := uuid.New()
	now := time.Now()

	pitchQuery := `
		INSERT INTO pitches (
			id, user_id, content, language, main_category, length_category,
			created_at, updated_at, posted_by, author_type, author_name, author_handle
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = tx.ExecContext(ctx, pitchQuery,
		pitchID,
		userID,
		pitchData.Content,
		pitchData.Language,
		pitchData.MainCategory,
		pitchData.LengthCategory,
		now,
		now,
		userID, // posted_by same as user_id
		pitchData.AuthorType,
		pitchData.AuthorName,
		pitchData.AuthorHandle,
	)

	if err != nil {
		return fmt.Errorf("failed to insert pitch: %w", err)
	}

	// Create tags
	for _, tagName := range pitchData.Tags {
		// Insert or get tag
		var tagID uuid.UUID
		tagQuery := `
			INSERT INTO tags (id, name, usage_count, created_at, updated_at)
			VALUES ($1, $2, 1, $3, $3)
			ON CONFLICT (name) DO UPDATE
			SET usage_count = tags.usage_count + 1,
				updated_at = $3
			RETURNING id
		`

		tagUUID := uuid.New()
		err = tx.GetContext(ctx, &tagID, tagQuery, tagUUID, tagName, now)
		if err != nil {
			return fmt.Errorf("failed to upsert tag %s: %w", tagName, err)
		}

		// Create pitch-tag relationship
		pitchTagID := uuid.New()
		pitchTagQuery := `
			INSERT INTO pitch_tags (id, pitch_id, tag_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4)
		`
		_, err = tx.ExecContext(ctx, pitchTagQuery, pitchTagID, pitchID, tagID, now)
		if err != nil {
			return fmt.Errorf("failed to create pitch-tag relationship: %w", err)
		}
	}

	// Commit transaction
	return tx.Commit()
}

func stringPtr(s string) *string {
	return &s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// NpubToHex converts a npub (bech32) format to hex format
func NpubToHex(npub string) (string, error) {
	// Decode bech32
	hrp, data, err := bech32.Decode(npub)
	if err != nil {
		return "", fmt.Errorf("failed to decode bech32: %v", err)
	}

	if hrp != "npub" {
		return "", fmt.Errorf("invalid hrp, expected 'npub', got '%s'", hrp)
	}

	// Convert from 5-bit to 8-bit
	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", fmt.Errorf("failed to convert bits: %v", err)
	}

	if len(converted) != 32 {
		return "", fmt.Errorf("invalid pubkey length: %d", len(converted))
	}

	return hex.EncodeToString(converted), nil
}

// cleanupDuplicateNostrUsers removes duplicate nostr users, keeping the one with hex auth_id
func cleanupDuplicateNostrUsers(db *sqlx.DB) error {
	// Use the actual hex pubkey from the database
	hexPubkey := "3cb60326b156fad8b996dfae73bb820dbdde1f0ac23f76c85b78a9b9813b7b3e"
	npub := "npub18jmqxe93tawcmwvhkh4zwwpqmk70whvdxcemyd7xkwl7uy3asulst0w8ra" // Keep for reference

	// Start transaction
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find all nostr users for this pubkey
	var userIDs []string
	query := `SELECT id FROM users WHERE auth_type = 'nostr' AND (auth_id = $1 OR auth_id = $2)`
	err = tx.Select(&userIDs, query, hexPubkey, npub)
	if err != nil {
		return err
	}

	if len(userIDs) <= 1 {
		// No duplicates to clean up
		return tx.Commit()
	}

	fmt.Printf("Found %d duplicate nostr users, cleaning up...\n", len(userIDs))

	// Keep the user with hex auth_id, delete the others
	var deleteUserIDs []string

	for _, userID := range userIDs {
		var authID string
		err = tx.Get(&authID, `SELECT auth_id FROM users WHERE id = $1`, userID)
		if err != nil {
			return err
		}

		if authID == hexPubkey {
			fmt.Printf("Keeping user %s with hex auth_id\n", userID)
		} else {
			deleteUserIDs = append(deleteUserIDs, userID)
			fmt.Printf("Will delete user %s with npub auth_id\n", userID)
		}
	}

	// Delete pitches and votes for users to be deleted
	for _, userID := range deleteUserIDs {
		// Delete votes first
		_, err = tx.Exec(`DELETE FROM votes WHERE user_id = $1`, userID)
		if err != nil {
			return fmt.Errorf("failed to delete votes for user %s: %v", userID, err)
		}

		// Delete pitch_tags for pitches by this user
		_, err = tx.Exec(`DELETE FROM pitch_tags WHERE pitch_id IN (SELECT id FROM pitches WHERE user_id = $1)`, userID)
		if err != nil {
			return fmt.Errorf("failed to delete pitch_tags for user %s: %v", userID, err)
		}

		// Delete pitches
		_, err = tx.Exec(`DELETE FROM pitches WHERE user_id = $1`, userID)
		if err != nil {
			return fmt.Errorf("failed to delete pitches for user %s: %v", userID, err)
		}

		// Delete user
		_, err = tx.Exec(`DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			return fmt.Errorf("failed to delete user %s: %v", userID, err)
		}

		fmt.Printf("Deleted duplicate user %s\n", userID)
	}

	return tx.Commit()
}
