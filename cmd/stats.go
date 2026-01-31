package cmd

import (
	"fmt"

	"srs/internal/db"

	"github.com/spf13/cobra"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics from database",
	Long:  `Display due card count and new card count from database.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database (create if doesn't exist)
		database, err := openDatabase()
		if err != nil {
			handleError(err, "failed to open database")
		}
		defer db.CloseDB(database)

		// Get due count
		dueCount, err := db.GetDueCount(database)
		if err != nil {
			handleError(err, "failed to get due count")
		}

		// Get new count
		newCount, err := db.GetNewCount(database)
		if err != nil {
			handleError(err, "failed to get new count")
		}

		// Display statistics
		fmt.Printf("Due cards: %d\n", dueCount)
		fmt.Printf("New cards: %d\n", newCount)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
