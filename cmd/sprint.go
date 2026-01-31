package cmd

import (
	"fmt"
	"os"

	"srs/internal/config"
	"srs/internal/db"

	"github.com/spf13/cobra"
)

// sprintCmd represents the sprint command
var sprintCmd = &cobra.Command{
	Use:   "sprint",
	Short: "Start a 2-minute review session",
	Long:  `Start a quick 2-minute review session. This is an alias for 'review --minutes 2' with session summary.`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get cards path: %v\n", err)
			os.Exit(1)
		}

		// Get tag filters from flags (shared with review command)
		// Run review session with 2-minute limit
		stats, err := runReviewSession(database, cardsPath, 0, 2, tagFlags, excludeTagFlags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
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
