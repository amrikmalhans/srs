package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"srs/internal/config"
	"srs/internal/db"
	"srs/internal/domain"
	"srs/internal/scheduler"
	"srs/internal/store"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync database from filesystem",
	Long:  `Scan markdown cards and update database metadata and review state.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get cards path: %v\n", err)
			os.Exit(1)
		}

		// Get config path and database path
		configPath, err := config.ResolveConfigPath("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to resolve config path: %v\n", err)
			os.Exit(1)
		}

		dbPath := config.DatabasePath(configPath)

		// Open database
		database, err := db.OpenDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
			os.Exit(1)
		}
		defer db.CloseDB(database)

		// Get all existing card IDs from database
		existingCardIDs, err := db.GetAllCardIDs(database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get existing card IDs: %v\n", err)
			os.Exit(1)
		}

		// Create a map for quick lookup
		existingCardMap := make(map[uuid.UUID]bool)
		for _, id := range existingCardIDs {
			existingCardMap[id] = true
		}

		// Scan all markdown files
		filePaths, err := store.ScanCardFiles(cardsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to scan card files: %v\n", err)
			os.Exit(1)
		}

		// Process each file
		var processedCount int
		var newCount int
		var errors []string

		for _, filePath := range filePaths {
			// Parse card from file
			card, err := store.ParseCardFile(filePath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to parse %s: %v", filePath, err))
				continue
			}

			// Get absolute path for storage
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to get absolute path for %s: %v", filePath, err))
				continue
			}

			// Upsert card metadata
			meta := &domain.CardMeta{
				CardID:    card.ID,
				Path:      absPath,
				UpdatedAt: time.Now(),
			}

			if err := db.UpsertCardMeta(database, meta); err != nil {
				errors = append(errors, fmt.Sprintf("failed to upsert card meta for %s: %v", card.ID.String(), err))
				continue
			}

			// Check if card is new (not in database)
			isNew := !existingCardMap[card.ID]

			if isNew {
				// Create default review state for new cards
				reviewState := &domain.ReviewState{
					CardID:         card.ID,
					DueAt:          time.Now(), // Due immediately
					IntervalDays:   0,
					Ease:           scheduler.DefaultEase, // Default SM-2 ease factor
					Reps:           0,
					Lapses:         0,
					LastReviewedAt: nil,
				}

				if err := db.UpsertReviewState(database, reviewState); err != nil {
					errors = append(errors, fmt.Sprintf("failed to create review state for %s: %v", card.ID.String(), err))
					continue
				}

				newCount++
			}

			processedCount++
		}

		// Print results
		fmt.Printf("Synced %d cards", processedCount)
		if newCount > 0 {
			fmt.Printf(" (%d new)", newCount)
		}
		fmt.Println()

		// Print errors if any
		if len(errors) > 0 {
			fmt.Fprintf(os.Stderr, "\nErrors encountered:\n")
			for _, errMsg := range errors {
				fmt.Fprintf(os.Stderr, "  %s\n", errMsg)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
