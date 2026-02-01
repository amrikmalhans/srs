package cmd

import (
	"fmt"

	"github.com/amrikmalhans/srs/internal/config"
	"github.com/amrikmalhans/srs/internal/db"

	"github.com/spf13/cobra"
)

// sprintCmd represents the sprint command
var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Start a 2-minute review session",
	Long:  `Start a quick 2-minute review session. This is an alias for 'review --minutes 2' with session summary.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database
		database, err := openDatabase()
		if err != nil {
			handleError(err, "failed to open database")
		}
		defer db.CloseDB(database)

		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			handleError(err, "failed to get cards path")
		}

		// Get tag filters from flags (shared with review command)
		// Run review session with default sprint duration
		stats, err := runReviewSession(database, cardsPath, UnlimitedCount, DefaultSprintMinutes, tagFlags, excludeTagFlags)
		if err != nil {
			handleError(err, "review session failed")
		}

		// Always display summary for sprint
		if stats.Reviewed > 0 {
			displaySessionSummary(stats)
		} else {
			fmt.Println("No cards due for review.")
			// Still show summary even if no cards reviewed
			displaySessionSummary(stats)
		}
	},
}

func init() {
	rootCmd.AddCommand(sprintCmd)

	// Add tag filter flags (shared with review command)
	sprintCmd.Flags().StringArrayVar(&tagFlags, "tag", []string{}, "Include only cards with this tag (can be repeated)")
	sprintCmd.Flags().StringArrayVar(&excludeTagFlags, "exclude-tag", []string{}, "Exclude cards with this tag (can be repeated)")
}
