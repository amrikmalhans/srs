package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/amrikmalhans/srs/internal/db"
	"github.com/amrikmalhans/srs/internal/domain"
	"github.com/amrikmalhans/srs/internal/scheduler"
	"github.com/amrikmalhans/srs/internal/store"

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
		cardsPath, err := getCardsPath()
		if err != nil {
			handleError(err, "failed to get cards path")
		}

		// Open database
		database, err := openDatabase()
		if err != nil {
			handleError(err, "failed to open database")
		}
		defer db.CloseDB(database)

		// Get all existing card IDs from database
		existingCardIDs, err := db.GetAllCardIDs(database)
		if err != nil {
			handleError(err, "failed to get existing card IDs")
		}

		// Create a map for quick lookup
		existingCardMap := make(map[uuid.UUID]bool)
		for _, id := range existingCardIDs {
			existingCardMap[id] = true
		}

		// Get all existing review state card IDs to check which cards need review states
		existingReviewStates, err := db.GetAllReviewStateCardIDs(database)
		if err != nil {
			handleError(err, "failed to get existing review state card IDs")
		}

		// Create a map for quick lookup
		existingReviewStateMap := make(map[uuid.UUID]bool)
		for _, id := range existingReviewStates {
			existingReviewStateMap[id] = true
		}

		// Scan all markdown files
		filePaths, err := store.ScanCardFiles(cardsPath)
		if err != nil {
			handleError(err, "failed to scan card files")
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

			// Check if review state exists (card might exist but review state was reset)
			needsReviewState := !existingReviewStateMap[card.ID]

			if isNew {
				newCount++
			}

			if isNew || needsReviewState {
				// Create default review state for new cards or cards missing review state
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
