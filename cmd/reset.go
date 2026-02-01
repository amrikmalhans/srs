package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/amrikmalhans/srs/internal/db"
	"github.com/amrikmalhans/srs/internal/ui"

	"github.com/spf13/cobra"
)

var forceFlag bool

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all review statistics and scheduling data",
	Long: `Reset all review statistics and scheduling data.

This command deletes all review state records from the database, effectively
resetting all cards to a "new" state. All scheduling information including
due dates, intervals, ease factors, repetitions, and lapses will be cleared.

The card files themselves are not affected - only the review statistics are reset.

Use --force to skip the confirmation prompt.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database
		database, err := openDatabase()
		if err != nil {
			handleError(err, "failed to open database")
		}
		defer db.CloseDB(database)

		// Get current count of review states
		count, err := db.GetReviewStateCount(database)
		if err != nil {
			handleError(err, "failed to get review state count")
		}

		if count == 0 {
			fmt.Println("No review statistics to reset.")
			return
		}

		// Confirm before proceeding (unless --force is used)
		if !forceFlag {
			fmt.Printf("%s\n", ui.ColorSummary("--- Reset Review Statistics ---"))
			fmt.Printf("This will delete all review statistics for %d card(s).\n", count)
			fmt.Printf("All scheduling data (due dates, intervals, ease factors, etc.) will be cleared.\n")
			fmt.Printf("Card files will not be affected.\n\n")
			fmt.Print(ui.ColorPrompt("Are you sure you want to continue? (yes/no): "))

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				handleError(err, "failed to read confirmation")
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "yes" && response != "y" {
				fmt.Println("Reset cancelled.")
				return
			}
		}

		// Delete all review states
		err = db.DeleteAllReviewStates(database)
		if err != nil {
			handleError(err, "failed to reset review statistics")
		}

		fmt.Printf("%s\n", ui.ColorSummary("Review statistics reset successfully."))
		fmt.Printf("All %d card(s) are now in a new state and ready for review.\n", count)
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)

	// Add flags
	resetCmd.Flags().BoolVar(&forceFlag, "force", false, "Skip confirmation prompt")
}
