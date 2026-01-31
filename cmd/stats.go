package cmd

import (
	"fmt"
	"os"

	"srs/internal/config"
	"srs/internal/db"

	"github.com/spf13/cobra"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show statistics from database",
	Long:  `Display due card count and new card count from database.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get config path and database path
		configPath, err := config.ResolveConfigPath("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to resolve config path: %v\n", err)
			os.Exit(1)
		}

		dbPath := config.DatabasePath(configPath)

		// Open database (create if doesn't exist)
		database, err := db.OpenDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
			os.Exit(1)
		}
		defer db.CloseDB(database)

		// Get due count
		dueCount, err := db.GetDueCount(database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get due count: %v\n", err)
			os.Exit(1)
		}

		// Get new count
		newCount, err := db.GetNewCount(database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get new count: %v\n", err)
			os.Exit(1)
		}

		// Display statistics
		fmt.Printf("Due cards: %d\n", dueCount)
		fmt.Printf("New cards: %d\n", newCount)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
