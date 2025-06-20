package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"bitcoinpitch.org/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

// Repository handles database operations for all models
type Repository struct {
	db *DB
}

// NewRepository creates a new repository
func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

// Ping checks if the database connection is alive
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// User operations

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (
			id, auth_type, auth_id, username, display_name, created_at, updated_at,
			show_auth_method, show_username, show_profile_info,
			email, password_hash, email_verified, email_verification_token, email_verification_expires_at,
			role, totp_secret, totp_enabled, totp_backup_codes,
			password_reset_token, password_reset_expires_at, page_size
		)
		VALUES (
			:id, :auth_type, :auth_id, :username, :display_name, :created_at, :updated_at,
			:show_auth_method, :show_username, :show_profile_info,
			:email, :password_hash, :email_verified, :email_verification_token, :email_verification_expires_at,
			:role, :totp_secret, :totp_enabled, :totp_backup_codes,
			:password_reset_token, :password_reset_expires_at, :page_size
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

// GetUserByAuth gets a user by their auth type and ID
func (r *Repository) GetUserByAuth(ctx context.Context, authType models.AuthType, authID string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE auth_type = $1 AND auth_id = $2`
	err := r.db.GetContext(ctx, &user, query, authType, authID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID gets a user by their ID
func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (r *Repository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = :username, display_name = :display_name, updated_at = :updated_at, 
		    show_auth_method = :show_auth_method, show_username = :show_username, show_profile_info = :show_profile_info,
		    email = :email, password_hash = :password_hash, email_verified = :email_verified,
		    email_verification_token = :email_verification_token, email_verification_expires_at = :email_verification_expires_at,
		    role = :role, totp_secret = :totp_secret, totp_enabled = :totp_enabled, totp_backup_codes = :totp_backup_codes,
		    password_reset_token = :password_reset_token, password_reset_expires_at = :password_reset_expires_at,
		    page_size = :page_size
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

// Session operations

// CreateSession creates a new session
func (r *Repository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at, updated_at)
		VALUES (:id, :user_id, :token, :expires_at, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, session)
	return err
}

// GetSessionByToken gets a session by its token
func (r *Repository) GetSessionByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	query := `SELECT * FROM sessions WHERE token = $1`
	err := r.db.GetContext(ctx, &session, query, token)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession deletes a session
func (r *Repository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteExpiredSessions deletes all expired sessions
func (r *Repository) DeleteExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM sessions WHERE expires_at < $1`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	return err
}

// Pitch operations

// CreatePitch creates a new pitch
func (r *Repository) CreatePitch(ctx context.Context, pitch *models.Pitch) error {
	return r.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Insert pitch
		query := `
			INSERT INTO pitches (
				id, user_id, content, language, main_category, length_category,
				created_at, updated_at, posted_by, author_type, author_name, author_handle
			)
			VALUES (
				:id, :user_id, :content, :language, :main_category, :length_category,
				:created_at, :updated_at, :posted_by, :author_type, :author_name, :author_handle
			)
		`
		_, err := tx.NamedExecContext(ctx, query, pitch)
		if err != nil {
			return fmt.Errorf("error creating pitch: %w", err)
		}

		// Insert tags if any
		if len(pitch.Tags) > 0 {
			for _, tag := range pitch.Tags {
				// Insert or update tag
				tagQuery := `
					INSERT INTO tags (id, name, usage_count, created_at, updated_at)
					VALUES (:id, :name, :usage_count, :created_at, :updated_at)
					ON CONFLICT (name) DO UPDATE
					SET usage_count = tags.usage_count + 1,
						updated_at = :updated_at
				`
				_, err := tx.NamedExecContext(ctx, tagQuery, tag)
				if err != nil {
					return fmt.Errorf("error upserting tag: %w", err)
				}

				// Fetch tag ID by name
				var tagID uuid.UUID
				getTagIDQuery := `SELECT id FROM tags WHERE name = $1`
				err = tx.GetContext(ctx, &tagID, getTagIDQuery, tag.Name)
				if err != nil {
					return fmt.Errorf("error fetching tag id: %w", err)
				}

				// Create pitch-tag relationship
				pitchTag := models.PitchTag{
					PitchID: pitch.ID,
					TagID:   tagID,
					BaseModel: models.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}
				pitchTagQuery := `
					INSERT INTO pitch_tags (pitch_id, tag_id, id, created_at, updated_at)
					VALUES (:pitch_id, :tag_id, :id, :created_at, :updated_at)
				`
				_, err = tx.NamedExecContext(ctx, pitchTagQuery, pitchTag)
				if err != nil {
					return fmt.Errorf("error creating pitch-tag relationship: %w", err)
				}
			}
		}

		return nil
	})
}

// GetPitch gets a pitch by ID
func (r *Repository) GetPitch(ctx context.Context, id uuid.UUID) (*models.Pitch, error) {
	var pitch models.Pitch
	query := `
		SELECT p.*, 
		       u.display_name as posted_by_display_name,
		       u.auth_type as posted_by_auth_type,
		       u.username as posted_by_username,
		       COALESCE(json_agg(jsonb_build_object(
		         'id', t.id,
		         'name', t.name,
		         'usage_count', t.usage_count,
		         'created_at', t.created_at,
		         'updated_at', t.updated_at
		       )) FILTER (WHERE t.id IS NOT NULL), '[]') AS tags
		FROM pitches p
		LEFT JOIN users u ON p.posted_by = u.id
		LEFT JOIN pitch_tags pt ON p.id = pt.pitch_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		WHERE p.id = $1 AND p.deleted_at IS NULL
		GROUP BY p.id, u.display_name, u.auth_type, u.username
	`
	err := r.db.GetContext(ctx, &pitch, query, id)
	if err != nil {
		return nil, err
	}
	return &pitch, nil
}

// UpdatePitch updates a pitch
func (r *Repository) UpdatePitch(ctx context.Context, pitch *models.Pitch) error {
	return r.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Update pitch
		query := `
			UPDATE pitches
			SET content = :content,
				language = :language,
				main_category = :main_category,
				length_category = :length_category,
				updated_at = :updated_at,
				last_edit_at = :last_edit_at,
				author_type = :author_type,
				author_name = :author_name,
				author_handle = :author_handle,
				vote_count = :vote_count,
				upvote_count = :upvote_count,
				downvote_count = :downvote_count,
				score = :score,
				last_vote_at = :last_vote_at
			WHERE id = :id AND deleted_at IS NULL
		`
		_, err := tx.NamedExecContext(ctx, query, pitch)
		if err != nil {
			return fmt.Errorf("error updating pitch: %w", err)
		}

		// Update tags if any
		if len(pitch.Tags) > 0 {
			// Delete existing tags
			_, err = tx.ExecContext(ctx, "DELETE FROM pitch_tags WHERE pitch_id = $1", pitch.ID)
			if err != nil {
				return fmt.Errorf("error deleting existing tags: %w", err)
			}

			// Insert new tags
			for _, tag := range pitch.Tags {
				// Insert or update tag
				tagQuery := `
					INSERT INTO tags (id, name, usage_count, created_at, updated_at)
					VALUES (:id, :name, :usage_count, :created_at, :updated_at)
					ON CONFLICT (name) DO UPDATE
					SET usage_count = tags.usage_count + 1,
						updated_at = :updated_at
				`
				_, err := tx.NamedExecContext(ctx, tagQuery, tag)
				if err != nil {
					return fmt.Errorf("error upserting tag: %w", err)
				}

				// Fetch tag ID by name
				var tagID uuid.UUID
				getTagIDQuery := `SELECT id FROM tags WHERE name = $1`
				err = tx.GetContext(ctx, &tagID, getTagIDQuery, tag.Name)
				if err != nil {
					return fmt.Errorf("error fetching tag id: %w", err)
				}

				// Create pitch-tag relationship
				pitchTag := models.PitchTag{
					PitchID: pitch.ID,
					TagID:   tagID,
					BaseModel: models.BaseModel{
						ID:        uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}
				pitchTagQuery := `
					INSERT INTO pitch_tags (pitch_id, tag_id, id, created_at, updated_at)
					VALUES (:pitch_id, :tag_id, :id, :created_at, :updated_at)
				`
				_, err = tx.NamedExecContext(ctx, pitchTagQuery, pitchTag)
				if err != nil {
					return fmt.Errorf("error creating pitch-tag relationship: %w", err)
				}
			}
		}

		return nil
	})
}

// DeletePitch soft deletes a pitch
func (r *Repository) DeletePitch(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE pitches
		SET deleted_at = $1,
			updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// ListPitches lists pitches with optional filters
func (r *Repository) ListPitches(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Pitch, error) {
	query := `
		SELECT p.*, 
		       u.display_name as posted_by_display_name,
		       u.auth_type as posted_by_auth_type,
		       u.username as posted_by_username,
		       u.show_auth_method as posted_by_show_auth_method,
		       u.show_username as posted_by_show_username,
		       u.show_profile_info as posted_by_show_profile_info,
		       COALESCE(json_agg(jsonb_build_object(
		         'id', t.id,
		         'name', t.name,
		         'usage_count', t.usage_count,
		         'created_at', t.created_at,
		         'updated_at', t.updated_at
		       )) FILTER (WHERE t.id IS NOT NULL), '[]') AS tags
		FROM pitches p
		LEFT JOIN users u ON p.posted_by = u.id
		LEFT JOIN pitch_tags pt ON p.id = pt.pitch_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		WHERE p.deleted_at IS NULL
	`
	args := []interface{}{}
	argCount := 1

	// Add filters with explicit column mapping for safety
	if len(filters) > 0 {
		for key, value := range filters {
			switch key {
			case "main_category":
				query += fmt.Sprintf(" AND p.main_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "language":
				query += fmt.Sprintf(" AND p.language = $%d", argCount)
				args = append(args, value)
				argCount++
			case "length_category":
				query += fmt.Sprintf(" AND p.length_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "user_id":
				query += fmt.Sprintf(" AND p.user_id = $%d", argCount)
				args = append(args, value)
				argCount++
			}
		}
	}

	query += `
		GROUP BY p.id, u.display_name, u.auth_type, u.username, u.show_auth_method, u.show_username, u.show_profile_info
		ORDER BY p.score DESC, p.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount) + `
		OFFSET $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit, offset)

	var pitches []*models.Pitch
	err := r.db.SelectContext(ctx, &pitches, query, args...)
	if err != nil {
		return nil, err
	}
	return pitches, nil
}

// CountPitches counts pitches with optional filters
func (r *Repository) CountPitches(ctx context.Context, filters map[string]interface{}) (int, error) {
	query := `SELECT COUNT(DISTINCT p.id) FROM pitches p WHERE p.deleted_at IS NULL`
	args := []interface{}{}
	argCount := 1

	// Add filters with explicit column mapping for safety
	if len(filters) > 0 {
		for key, value := range filters {
			switch key {
			case "main_category":
				query += fmt.Sprintf(" AND p.main_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "language":
				query += fmt.Sprintf(" AND p.language = $%d", argCount)
				args = append(args, value)
				argCount++
			case "length_category":
				query += fmt.Sprintf(" AND p.length_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "user_id":
				query += fmt.Sprintf(" AND p.user_id = $%d", argCount)
				args = append(args, value)
				argCount++
			}
		}
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Vote operations

// CreateVote creates a new vote
func (r *Repository) CreateVote(ctx context.Context, vote *models.Vote) error {
	return r.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Insert vote
		query := `
			INSERT INTO votes (id, pitch_id, user_id, vote_type, created_at, updated_at)
			VALUES (:id, :pitch_id, :user_id, :vote_type, :created_at, :updated_at)
		`
		_, err := tx.NamedExecContext(ctx, query, vote)
		if err != nil {
			return fmt.Errorf("error creating vote: %w", err)
		}

		// Update pitch vote counts
		updateQuery := `
			UPDATE pitches
			SET vote_count = vote_count + 1,
				upvote_count = CASE WHEN $1 = 'up' THEN upvote_count + 1 ELSE upvote_count END,
				downvote_count = CASE WHEN $1 = 'down' THEN downvote_count + 1 ELSE downvote_count END,
				score = CASE WHEN $1 = 'up' THEN score + 1 ELSE score - 1 END,
				last_vote_at = $2,
				updated_at = $2
			WHERE id = $3 AND deleted_at IS NULL
		`
		_, err = tx.ExecContext(ctx, updateQuery, vote.VoteType, time.Now(), vote.PitchID)
		if err != nil {
			return fmt.Errorf("error updating pitch vote counts: %w", err)
		}

		return nil
	})
}

// GetVote gets a vote by pitch ID and user ID
func (r *Repository) GetVote(ctx context.Context, pitchID, userID uuid.UUID) (*models.Vote, error) {
	var vote models.Vote
	query := `SELECT * FROM votes WHERE pitch_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &vote, query, pitchID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &vote, nil
}

// DeleteVote deletes a vote and updates pitch vote counts
func (r *Repository) DeleteVote(ctx context.Context, vote *models.Vote) error {
	return r.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Delete vote
		query := `DELETE FROM votes WHERE id = $1`
		_, err := tx.ExecContext(ctx, query, vote.ID)
		if err != nil {
			return fmt.Errorf("error deleting vote: %w", err)
		}

		// Update pitch vote counts
		updateQuery := `
			UPDATE pitches
			SET vote_count = vote_count - 1,
				upvote_count = CASE WHEN $1 = 'up' THEN upvote_count - 1 ELSE upvote_count END,
				downvote_count = CASE WHEN $1 = 'down' THEN downvote_count - 1 ELSE downvote_count END,
				score = CASE WHEN $1 = 'up' THEN score - 1 ELSE score + 1 END,
				updated_at = $2
			WHERE id = $3 AND deleted_at IS NULL
		`
		_, err = tx.ExecContext(ctx, updateQuery, vote.VoteType, time.Now(), vote.PitchID)
		if err != nil {
			return fmt.Errorf("error updating pitch vote counts: %w", err)
		}

		return nil
	})
}

// UpdateVote updates a vote and updates pitch vote counts
func (r *Repository) UpdateVote(ctx context.Context, vote *models.Vote) error {
	return r.db.WithTx(ctx, func(tx *sqlx.Tx) error {
		// Update vote
		query := `
			UPDATE votes
			SET vote_type = :vote_type,
				updated_at = :updated_at
			WHERE id = :id
		`
		_, err := tx.NamedExecContext(ctx, query, vote)
		if err != nil {
			return fmt.Errorf("error updating vote: %w", err)
		}

		// Update pitch vote counts
		updateQuery := `
			UPDATE pitches
			SET upvote_count = CASE WHEN $1 = 'up' THEN upvote_count + 1 ELSE upvote_count - 1 END,
				downvote_count = CASE WHEN $1 = 'down' THEN downvote_count + 1 ELSE downvote_count - 1 END,
				score = CASE WHEN $1 = 'up' THEN score + 2 ELSE score - 2 END,
				last_vote_at = $2,
				updated_at = $2
			WHERE id = $3 AND deleted_at IS NULL
		`
		_, err = tx.ExecContext(ctx, updateQuery, vote.VoteType, time.Now(), vote.PitchID)
		if err != nil {
			return fmt.Errorf("error updating pitch vote counts: %w", err)
		}

		return nil
	})
}

// Tag operations

// ListTags lists tags with optional filters
func (r *Repository) ListTags(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Tag, error) {
	query := `SELECT * FROM tags`
	args := []interface{}{}
	argCount := 1

	// Add filters with explicit column mapping for safety
	if len(filters) > 0 {
		query += " WHERE"
		for key, value := range filters {
			switch key {
			case "name":
				query += fmt.Sprintf(" name = $%d AND", argCount)
				args = append(args, value)
				argCount++
			case "usage_count":
				query += fmt.Sprintf(" usage_count = $%d AND", argCount)
				args = append(args, value)
				argCount++
			}
		}
		if len(args) > 0 {
			query = query[:len(query)-4] // Remove trailing " AND"
		} else {
			query = query[:len(query)-6] // Remove " WHERE"
		}
	}

	query += `
		ORDER BY usage_count DESC, name ASC
		LIMIT $` + fmt.Sprintf("%d", argCount) + `
		OFFSET $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit, offset)

	var tags []*models.Tag
	err := r.db.SelectContext(ctx, &tags, query, args...)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// SearchTags searches tags by name
func (r *Repository) SearchTags(ctx context.Context, query string, limit int) ([]*models.Tag, error) {
	sqlQuery := `
		SELECT * FROM tags
		WHERE name ILIKE $1
		ORDER BY usage_count DESC, name ASC
		LIMIT $2
	`
	var tags []*models.Tag
	err := r.db.SelectContext(ctx, &tags, sqlQuery, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// ListPitchesByTag lists pitches filtered by category and tag
func (r *Repository) ListPitchesByTag(ctx context.Context, category, tagName string, limit, offset int) ([]*models.Pitch, error) {
	query := `
		SELECT p.*, 
		       u.display_name as posted_by_display_name,
		       u.auth_type as posted_by_auth_type,
		       u.username as posted_by_username,
		       u.show_auth_method as posted_by_show_auth_method,
		       u.show_username as posted_by_show_username,
		       u.show_profile_info as posted_by_show_profile_info,
		       COALESCE(json_agg(jsonb_build_object(
		         'id', t.id,
		         'name', t.name,
		         'usage_count', t.usage_count,
		         'created_at', t.created_at,
		         'updated_at', t.updated_at
		       )) FILTER (WHERE t.id IS NOT NULL), '[]') AS tags
		FROM pitches p
		LEFT JOIN users u ON p.posted_by = u.id
		LEFT JOIN pitch_tags pt ON p.id = pt.pitch_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		WHERE p.deleted_at IS NULL 
		  AND p.main_category = $1
		  AND p.id IN (
			  SELECT DISTINCT pt2.pitch_id 
			  FROM pitch_tags pt2 
			  JOIN tags t2 ON pt2.tag_id = t2.id 
			  WHERE t2.name = $2
		  )
		GROUP BY p.id, u.display_name, u.auth_type, u.username, u.show_auth_method, u.show_username, u.show_profile_info
		ORDER BY p.score DESC, p.created_at DESC
		LIMIT $3 OFFSET $4
	`
	var pitches []*models.Pitch
	err := r.db.SelectContext(ctx, &pitches, query, category, tagName, limit, offset)
	if err != nil {
		return nil, err
	}
	return pitches, nil
}

// CountPitchesByTag counts pitches filtered by category and tag
func (r *Repository) CountPitchesByTag(ctx context.Context, category, tagName string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT p.id)
		FROM pitches p
		WHERE p.deleted_at IS NULL 
		  AND p.main_category = $1
		  AND p.id IN (
			  SELECT DISTINCT pt2.pitch_id 
			  FROM pitch_tags pt2 
			  JOIN tags t2 ON pt2.tag_id = t2.id 
			  WHERE t2.name = $2
		  )
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, category, tagName)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAvailableLanguages returns a list of available languages from pitches
func (r *Repository) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT language 
		FROM pitches 
		WHERE deleted_at IS NULL AND language IS NOT NULL AND language != ''
		ORDER BY language
	`
	var languages []string
	err := r.db.SelectContext(ctx, &languages, query)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

// GetAvailableLanguagesByCategory returns available languages for a specific category
func (r *Repository) GetAvailableLanguagesByCategory(ctx context.Context, category string) ([]string, error) {
	query := `
		SELECT DISTINCT language
		FROM pitches
		WHERE main_category = $1 
		  AND deleted_at IS NULL
		ORDER BY language
	`
	var languages []string
	err := r.db.SelectContext(ctx, &languages, query, category)
	if err != nil {
		return nil, err
	}
	return languages, nil
}

// GetLanguageUsage returns language usage statistics (count of pitches per language)
func (r *Repository) GetLanguageUsage(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT language, COUNT(*) as count
		FROM pitches
		WHERE deleted_at IS NULL
		GROUP BY language
		ORDER BY count DESC, language
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usage := make(map[string]int)
	for rows.Next() {
		var language string
		var count int
		if err := rows.Scan(&language, &count); err != nil {
			return nil, err
		}
		usage[language] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usage, nil
}

// GetUserByEmail gets a user by their email address
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmailVerificationToken gets a user by their email verification token
func (r *Repository) GetUserByEmailVerificationToken(ctx context.Context, token string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email_verification_token = $1`
	err := r.db.GetContext(ctx, &user, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByRole gets users by their role
func (r *Repository) GetUserByRole(ctx context.Context, role models.UserRole) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users WHERE role = $1`
	err := r.db.SelectContext(ctx, &users, query, role)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUsersByRole gets users by their role with pagination
func (r *Repository) GetUsersByRole(ctx context.Context, role models.UserRole, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users WHERE role = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &users, query, role, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// CountUsersByRole counts users by their role
func (r *Repository) CountUsersByRole(ctx context.Context, role models.UserRole) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE role = $1`
	err := r.db.GetContext(ctx, &count, query, role)
	return count, err
}

// GetAllUsers gets all users with pagination
func (r *Repository) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// CountAllUsers counts all users
func (r *Repository) CountAllUsers(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

// CreateEmailVerificationToken creates an email verification token
func (r *Repository) CreateEmailVerificationToken(ctx context.Context, token *models.EmailVerificationToken) error {
	query := `
		INSERT INTO email_verification_tokens (id, user_id, token, email, expires_at, used, created_at, updated_at)
		VALUES (:id, :user_id, :token, :email, :expires_at, :used, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, token)
	return err
}

// GetEmailVerificationToken gets an email verification token by token
func (r *Repository) GetEmailVerificationToken(ctx context.Context, token string) (*models.EmailVerificationToken, error) {
	var verificationToken models.EmailVerificationToken
	query := `SELECT * FROM email_verification_tokens WHERE token = $1`
	err := r.db.GetContext(ctx, &verificationToken, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &verificationToken, nil
}

// UpdateEmailVerificationToken updates an email verification token
func (r *Repository) UpdateEmailVerificationToken(ctx context.Context, token *models.EmailVerificationToken) error {
	query := `
		UPDATE email_verification_tokens
		SET user_id = :user_id, expires_at = :expires_at, updated_at = :updated_at
		WHERE token = :token
	`
	_, err := r.db.NamedExecContext(ctx, query, token)
	return err
}

// DeleteEmailVerificationToken deletes an email verification token
func (r *Repository) DeleteEmailVerificationToken(ctx context.Context, token string) error {
	query := `DELETE FROM email_verification_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// ListPitchesByTagAndFilters lists pitches filtered by category, tag, and additional filters
func (r *Repository) ListPitchesByTagAndFilters(ctx context.Context, category, tagName string, filters map[string]interface{}, limit, offset int) ([]*models.Pitch, error) {
	query := `
		SELECT p.*, 
		       u.display_name as posted_by_display_name,
		       u.auth_type as posted_by_auth_type,
		       u.username as posted_by_username,
		       u.show_auth_method as posted_by_show_auth_method,
		       u.show_username as posted_by_show_username,
		       u.show_profile_info as posted_by_show_profile_info,
		       COALESCE(json_agg(jsonb_build_object(
		         'id', t.id,
		         'name', t.name,
		         'usage_count', t.usage_count,
		         'created_at', t.created_at,
		         'updated_at', t.updated_at
		       )) FILTER (WHERE t.id IS NOT NULL), '[]') AS tags
		FROM pitches p
		LEFT JOIN users u ON p.posted_by = u.id
		LEFT JOIN pitch_tags pt ON p.id = pt.pitch_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		INNER JOIN pitch_tags pt2 ON p.id = pt2.pitch_id
		INNER JOIN tags t2 ON pt2.tag_id = t2.id
		WHERE p.deleted_at IS NULL 
		  AND p.main_category = $1 
		  AND t2.name = $2
	`
	args := []interface{}{category, tagName}
	argCount := 3

	// Add additional filters with explicit column mapping for safety
	if len(filters) > 0 {
		for key, value := range filters {
			switch key {
			case "language":
				query += fmt.Sprintf(" AND p.language = $%d", argCount)
				args = append(args, value)
				argCount++
			case "length_category":
				query += fmt.Sprintf(" AND p.length_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "user_id":
				query += fmt.Sprintf(" AND p.user_id = $%d", argCount)
				args = append(args, value)
				argCount++
			}
		}
	}

	query += `
		GROUP BY p.id, u.display_name, u.auth_type, u.username, u.show_auth_method, u.show_username, u.show_profile_info
		ORDER BY p.score DESC, p.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount) + `
		OFFSET $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit, offset)

	var pitches []*models.Pitch
	err := r.db.SelectContext(ctx, &pitches, query, args...)
	if err != nil {
		return nil, err
	}
	return pitches, nil
}

// CountPitchesByTagAndFilters counts pitches filtered by category, tag, and additional filters
func (r *Repository) CountPitchesByTagAndFilters(ctx context.Context, category, tagName string, filters map[string]interface{}) (int, error) {
	query := `
		SELECT COUNT(DISTINCT p.id)
		FROM pitches p
		INNER JOIN pitch_tags pt2 ON p.id = pt2.pitch_id
		INNER JOIN tags t2 ON pt2.tag_id = t2.id
		WHERE p.deleted_at IS NULL 
		  AND p.main_category = $1 
		  AND t2.name = $2
	`
	args := []interface{}{category, tagName}
	argCount := 3

	// Add additional filters with explicit column mapping for safety
	if len(filters) > 0 {
		for key, value := range filters {
			switch key {
			case "language":
				query += fmt.Sprintf(" AND p.language = $%d", argCount)
				args = append(args, value)
				argCount++
			case "length_category":
				query += fmt.Sprintf(" AND p.length_category = $%d", argCount)
				args = append(args, value)
				argCount++
			case "user_id":
				query += fmt.Sprintf(" AND p.user_id = $%d", argCount)
				args = append(args, value)
				argCount++
			}
		}
	}

	var count int
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetConfigSetting retrieves a configuration setting by key
func (r *Repository) GetConfigSetting(ctx context.Context, key string) (*models.ConfigSetting, error) {
	var setting models.ConfigSetting
	query := `SELECT * FROM config_settings WHERE key = $1`
	err := r.db.GetContext(ctx, &setting, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &setting, nil
}

// GetConfigSettingsByCategory retrieves all configuration settings in a category
func (r *Repository) GetConfigSettingsByCategory(ctx context.Context, category string) ([]*models.ConfigSetting, error) {
	var settings []*models.ConfigSetting
	query := `SELECT * FROM config_settings WHERE category = $1 ORDER BY key`
	err := r.db.SelectContext(ctx, &settings, query, category)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// GetAllConfigSettings retrieves all configuration settings
func (r *Repository) GetAllConfigSettings(ctx context.Context) ([]*models.ConfigSetting, error) {
	var settings []*models.ConfigSetting
	query := `SELECT * FROM config_settings ORDER BY category, key`
	err := r.db.SelectContext(ctx, &settings, query)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

// CreateConfigSetting creates a new configuration setting
func (r *Repository) CreateConfigSetting(ctx context.Context, setting *models.ConfigSetting) error {
	query := `
		INSERT INTO config_settings (id, key, value, description, category, data_type, created_at, updated_at, updated_by)
		VALUES (:id, :key, :value, :description, :category, :data_type, :created_at, :updated_at, :updated_by)
	`
	_, err := r.db.NamedExecContext(ctx, query, setting)
	return err
}

// UpdateConfigSetting updates an existing configuration setting
func (r *Repository) UpdateConfigSetting(ctx context.Context, setting *models.ConfigSetting) error {
	query := `
		UPDATE config_settings 
		SET value = :value, description = :description, category = :category, 
		    data_type = :data_type, updated_at = :updated_at, updated_by = :updated_by
		WHERE key = :key
	`
	result, err := r.db.NamedExecContext(ctx, query, setting)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteConfigSetting deletes a configuration setting
func (r *Repository) DeleteConfigSetting(ctx context.Context, key string) error {
	query := `DELETE FROM config_settings WHERE key = $1`
	result, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// CreateConfigAuditLog creates a new configuration audit log entry
func (r *Repository) CreateConfigAuditLog(ctx context.Context, log *models.ConfigAuditLog) error {
	query := `
		INSERT INTO config_audit_log (id, config_key, old_value, new_value, changed_by, changed_at, action, created_at, updated_at)
		VALUES (:id, :config_key, :old_value, :new_value, :changed_by, :changed_at, :action, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, log)
	return err
}

// GetConfigAuditLogs retrieves audit logs for a configuration key
func (r *Repository) GetConfigAuditLogs(ctx context.Context, configKey string, limit, offset int) ([]*models.ConfigAuditLog, error) {
	var logs []*models.ConfigAuditLog
	query := `
		SELECT cal.*, u.display_name as changed_by_name, u.username as changed_by_username
		FROM config_audit_log cal
		LEFT JOIN users u ON cal.changed_by = u.id
		WHERE cal.config_key = $1
		ORDER BY cal.changed_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &logs, query, configKey, limit, offset)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetAllConfigAuditLogs retrieves all audit logs with pagination
func (r *Repository) GetAllConfigAuditLogs(ctx context.Context, limit, offset int) ([]*models.ConfigAuditLog, error) {
	var logs []*models.ConfigAuditLog
	query := `
		SELECT 
			cal.id,
			cal.changed_at as created_at,
			cal.changed_at as updated_at,
			cal.config_key,
			cal.old_value,
			cal.new_value,
			cal.changed_by,
			cal.changed_at,
			cal.action,
			u.email as changed_by_email,
			u.username as changed_by_username,
			u.display_name as changed_by_display_name
		FROM config_audit_log cal
		LEFT JOIN users u ON cal.changed_by = u.id
		ORDER BY cal.changed_at DESC
		LIMIT $1 OFFSET $2
	`
	err := r.db.SelectContext(ctx, &logs, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// UserActivity methods
func (r *Repository) CreateUserActivity(ctx context.Context, activity *models.UserActivity) error {
	query := `
		INSERT INTO user_activities (id, user_id, action_type, target_id, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		activity.ID,
		activity.UserID,
		activity.ActionType,
		activity.TargetID,
		activity.IPAddress,
		activity.UserAgent,
		activity.Metadata,
		activity.CreatedAt,
	)
	return err
}

func (r *Repository) GetLastUserActivity(ctx context.Context, userID uuid.UUID, actionType models.ActivityType) (*models.UserActivity, error) {
	query := `
		SELECT id, user_id, action_type, target_id, ip_address, user_agent, metadata, created_at, created_at, created_at
		FROM user_activities 
		WHERE user_id = $1 AND action_type = $2 
		ORDER BY created_at DESC 
		LIMIT 1`

	var activity models.UserActivity
	err := r.db.QueryRowContext(ctx, query, userID, actionType).Scan(
		&activity.ID,
		&activity.UserID,
		&activity.ActionType,
		&activity.TargetID,
		&activity.IPAddress,
		&activity.UserAgent,
		&activity.Metadata,
		&activity.CreatedAt,
		&activity.UpdatedAt,
		&activity.CreatedAt, // Use created_at for updated_at as well
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &activity, nil
}

func (r *Repository) CountUserActivitiesSince(ctx context.Context, userID uuid.UUID, actionType models.ActivityType, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM user_activities 
		WHERE user_id = $1 AND action_type = $2 AND created_at >= $3`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, actionType, since).Scan(&count)
	return count, err
}

func (r *Repository) CountIPActivitiesSince(ctx context.Context, ipAddress net.IP, actionType models.ActivityType, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM user_activities 
		WHERE ip_address = $1 AND action_type = $2 AND created_at >= $3`

	var count int
	err := r.db.QueryRowContext(ctx, query, ipAddress, actionType, since).Scan(&count)
	return count, err
}

func (r *Repository) CleanupOldActivities(ctx context.Context, cutoff time.Time) error {
	query := `DELETE FROM user_activities WHERE created_at < $1`
	_, err := r.db.ExecContext(ctx, query, cutoff)
	return err
}

// UserPenalty methods
func (r *Repository) CreateUserPenalty(ctx context.Context, penalty *models.UserPenalty) error {
	query := `
		INSERT INTO user_penalties (id, user_id, penalty_type, reason, multiplier, expires_at, created_at, created_by, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		penalty.ID,
		penalty.UserID,
		penalty.PenaltyType,
		penalty.Reason,
		penalty.Multiplier,
		penalty.ExpiresAt,
		penalty.CreatedAt,
		penalty.CreatedBy,
		penalty.IsActive,
	)
	return err
}

func (r *Repository) GetActivePenaltiesForUser(ctx context.Context, userID uuid.UUID) ([]*models.UserPenalty, error) {
	query := `
		SELECT id, user_id, penalty_type, reason, multiplier, expires_at, created_at, created_at, created_by, is_active
		FROM user_penalties 
		WHERE user_id = $1 AND is_active = true AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var penalties []*models.UserPenalty
	for rows.Next() {
		var penalty models.UserPenalty
		err := rows.Scan(
			&penalty.ID,
			&penalty.UserID,
			&penalty.PenaltyType,
			&penalty.Reason,
			&penalty.Multiplier,
			&penalty.ExpiresAt,
			&penalty.CreatedAt,
			&penalty.UpdatedAt,
			&penalty.CreatedBy,
			&penalty.IsActive,
		)
		if err != nil {
			return nil, err
		}
		penalties = append(penalties, &penalty)
	}

	return penalties, rows.Err()
}

func (r *Repository) CleanupExpiredPenalties(ctx context.Context) error {
	query := `UPDATE user_penalties SET is_active = false WHERE expires_at <= NOW() AND is_active = true`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// ContentHash methods
func (r *Repository) CreateContentHash(ctx context.Context, contentHash *models.ContentHash) error {
	query := `
		INSERT INTO content_hashes (id, user_id, content_hash, original_content, pitch_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		contentHash.ID,
		contentHash.UserID,
		contentHash.ContentHash,
		contentHash.OriginalContent,
		contentHash.PitchID,
		contentHash.CreatedAt,
	)
	return err
}

func (r *Repository) GetContentHashesByHash(ctx context.Context, hash string) ([]*models.ContentHash, error) {
	query := `
		SELECT id, user_id, content_hash, original_content, pitch_id, created_at, created_at
		FROM content_hashes 
		WHERE content_hash = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []*models.ContentHash
	for rows.Next() {
		var ch models.ContentHash
		err := rows.Scan(
			&ch.ID,
			&ch.UserID,
			&ch.ContentHash,
			&ch.OriginalContent,
			&ch.PitchID,
			&ch.CreatedAt,
			&ch.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, &ch)
	}

	return hashes, rows.Err()
}
