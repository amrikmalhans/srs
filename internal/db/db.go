package db

import (
	"database/sql"
	"fmt"
	"time"

	"srs/internal/domain"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// OpenDB opens or creates a SQLite database at the given path
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Initialize schema
	if err := InitSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// InitSchema creates the database tables if they don't exist
func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS cards_meta (
		card_id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS review_state (
		card_id TEXT PRIMARY KEY,
		due_at TEXT NOT NULL,
		interval_days INTEGER NOT NULL,
		ease REAL NOT NULL,
		reps INTEGER NOT NULL,
		lapses INTEGER NOT NULL,
		last_reviewed_at TEXT
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB(db *sql.DB) error {
	return db.Close()
}

// UpsertCardMeta inserts or updates card metadata
func UpsertCardMeta(db *sql.DB, meta *domain.CardMeta) error {
	query := `
		INSERT INTO cards_meta (card_id, path, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(card_id) DO UPDATE SET
			path = excluded.path,
			updated_at = excluded.updated_at
	`

	_, err := db.Exec(query, meta.CardID.String(), meta.Path, meta.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to upsert card meta: %w", err)
	}

	return nil
}

// UpsertReviewState inserts or updates review state
func UpsertReviewState(db *sql.DB, state *domain.ReviewState) error {
	query := `
		INSERT INTO review_state (card_id, due_at, interval_days, ease, reps, lapses, last_reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(card_id) DO UPDATE SET
			due_at = excluded.due_at,
			interval_days = excluded.interval_days,
			ease = excluded.ease,
			reps = excluded.reps,
			lapses = excluded.lapses,
			last_reviewed_at = excluded.last_reviewed_at
	`

	var lastReviewedAt *string
	if state.LastReviewedAt != nil {
		formatted := state.LastReviewedAt.Format(time.RFC3339)
		lastReviewedAt = &formatted
	}

	_, err := db.Exec(
		query,
		state.CardID.String(),
		state.DueAt.Format(time.RFC3339),
		state.IntervalDays,
		state.Ease,
		state.Reps,
		state.Lapses,
		lastReviewedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert review state: %w", err)
	}

	return nil
}

// GetReviewState retrieves review state for a card
func GetReviewState(db *sql.DB, cardID uuid.UUID) (*domain.ReviewState, error) {
	query := `
		SELECT card_id, due_at, interval_days, ease, reps, lapses, last_reviewed_at
		FROM review_state
		WHERE card_id = ?
	`

	var state domain.ReviewState
	var dueAtStr, lastReviewedAtStr sql.NullString

	err := db.QueryRow(query, cardID.String()).Scan(
		&state.CardID,
		&dueAtStr,
		&state.IntervalDays,
		&state.Ease,
		&state.Reps,
		&state.Lapses,
		&lastReviewedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("review state not found for card: %s", cardID.String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get review state: %w", err)
	}

	// Parse due_at
	dueAt, err := time.Parse(time.RFC3339, dueAtStr.String)
	if err != nil {
		return nil, fmt.Errorf("failed to parse due_at: %w", err)
	}
	state.DueAt = dueAt

	// Parse last_reviewed_at if present
	if lastReviewedAtStr.Valid {
		lastReviewedAt, err := time.Parse(time.RFC3339, lastReviewedAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_reviewed_at: %w", err)
		}
		state.LastReviewedAt = &lastReviewedAt
	}

	return &state, nil
}

// GetAllCardIDs returns all card IDs from the database
func GetAllCardIDs(db *sql.DB) ([]uuid.UUID, error) {
	query := `SELECT card_id FROM cards_meta`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query card IDs: %w", err)
	}
	defer rows.Close()

	var cardIDs []uuid.UUID
	for rows.Next() {
		var cardIDStr string
		if err := rows.Scan(&cardIDStr); err != nil {
			return nil, fmt.Errorf("failed to scan card ID: %w", err)
		}

		cardID, err := uuid.Parse(cardIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse card ID: %w", err)
		}

		cardIDs = append(cardIDs, cardID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating card IDs: %w", err)
	}

	return cardIDs, nil
}

// GetDueCount returns the number of cards that are due for review
func GetDueCount(db *sql.DB) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM review_state 
		WHERE due_at <= ?
	`

	now := time.Now().Format(time.RFC3339)
	var count int
	err := db.QueryRow(query, now).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get due count: %w", err)
	}

	return count, nil
}

// GetNewCount returns the number of new cards (reps = 0 AND lapses = 0)
func GetNewCount(db *sql.DB) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM review_state 
		WHERE reps = 0 AND lapses = 0
	`

	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get new count: %w", err)
	}

	return count, nil
}

// GetDueCards returns all due cards ordered by due_at ASC
// If limit is <= 0, no limit is applied
func GetDueCards(db *sql.DB, limit int) ([]*domain.ReviewState, error) {
	now := time.Now().Format(time.RFC3339)

	var query string
	var args []interface{}

	if limit > 0 {
		query = `
			SELECT card_id, due_at, interval_days, ease, reps, lapses, last_reviewed_at
			FROM review_state
			WHERE due_at <= ?
			ORDER BY due_at ASC
			LIMIT ?
		`
		args = []interface{}{now, limit}
	} else {
		query = `
			SELECT card_id, due_at, interval_days, ease, reps, lapses, last_reviewed_at
			FROM review_state
			WHERE due_at <= ?
			ORDER BY due_at ASC
		`
		args = []interface{}{now}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query due cards: %w", err)
	}
	defer rows.Close()

	var states []*domain.ReviewState
	for rows.Next() {
		var state domain.ReviewState
		var cardIDStr, dueAtStr string
		var lastReviewedAtStr sql.NullString

		err := rows.Scan(
			&cardIDStr,
			&dueAtStr,
			&state.IntervalDays,
			&state.Ease,
			&state.Reps,
			&state.Lapses,
			&lastReviewedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review state: %w", err)
		}

		// Parse card ID
		cardID, err := uuid.Parse(cardIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse card ID: %w", err)
		}
		state.CardID = cardID

		// Parse due_at
		dueAt, err := time.Parse(time.RFC3339, dueAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse due_at: %w", err)
		}
		state.DueAt = dueAt

		// Parse last_reviewed_at if present
		if lastReviewedAtStr.Valid {
			lastReviewedAt, err := time.Parse(time.RFC3339, lastReviewedAtStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last_reviewed_at: %w", err)
			}
			state.LastReviewedAt = &lastReviewedAt
		}

		states = append(states, &state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating due cards: %w", err)
	}

	return states, nil
}

// GetNextDueTime returns the earliest due_at time for cards that are currently due.
// Returns nil if no cards are due, or an error if the query fails.
func GetNextDueTime(db *sql.DB) (*time.Time, error) {
	query := `
		SELECT MIN(due_at)
		FROM review_state
		WHERE due_at <= ?
	`

	now := time.Now().Format(time.RFC3339)
	var dueAtStr sql.NullString

	err := db.QueryRow(query, now).Scan(&dueAtStr)
	if err == sql.ErrNoRows || !dueAtStr.Valid {
		// No cards due
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next due time: %w", err)
	}

	// Parse due_at
	dueAt, err := time.Parse(time.RFC3339, dueAtStr.String)
	if err != nil {
		return nil, fmt.Errorf("failed to parse due_at: %w", err)
	}

	return &dueAt, nil
}
