package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/amrikmalhans/srs/internal/config"
	"github.com/amrikmalhans/srs/internal/db"
)

const (
	// DefaultSprintMinutes is the default duration for sprint review sessions
	DefaultSprintMinutes = 2

	// UnlimitedCount indicates no limit on the number of cards to review
	UnlimitedCount = 0
)

// openDatabase opens the database connection using the standard configuration path resolution.
// Returns the database connection and any error encountered.
func openDatabase() (*sql.DB, error) {
	configPath, err := config.ResolveConfigPath("")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	dbPath := config.DatabasePath(configPath)
	database, err := db.OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return database, nil
}

// handleError prints an error message to stderr and exits with code 1.
// This is used in command Run functions for consistent error handling.
func handleError(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", message, err)
		os.Exit(1)
	}
}
