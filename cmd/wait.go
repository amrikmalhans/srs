package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/amrikmalhans/srs/internal/db"

	"github.com/spf13/cobra"
)

// runSprintProgrammatically runs the sprint command programmatically
func runSprintProgrammatically() error {
	// Open database
	database, err := openDatabase()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.CloseDB(database)

	// Get cards path
	cardsPath, err := getCardsPath()
	if err != nil {
		return fmt.Errorf("failed to get cards path: %w", err)
	}

	// Run review session with default sprint duration (no tag filters)
	stats, err := runReviewSession(database, cardsPath, UnlimitedCount, DefaultSprintMinutes, []string{}, []string{})
	if err != nil {
		return fmt.Errorf("review session failed: %w", err)
	}

	// Always display summary for sprint
	if stats.Reviewed > 0 {
		displaySessionSummary(stats)
	} else {
		fmt.Println("No cards due for review.")
		// Still show summary even if no cards reviewed
		displaySessionSummary(stats)
	}

	return nil
}

// waitCmd represents the wait command
var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Execute a command and then run a sprint session",
	Long: `Execute a command and wait for it to complete, then automatically run 'srs sprint --minutes 2'.
This is useful for running reviews during CI/build dead time.

Example:
  srs wait "make build"
  srs wait "npm test"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commandStr := args[0]

		// Validate command string
		if commandStr == "" {
			fmt.Fprintf(os.Stderr, "Error: command cannot be empty\n")
			os.Exit(1)
		}
		if len(commandStr) > 10000 {
			fmt.Fprintf(os.Stderr, "Error: command too long (max 10000 characters)\n")
			os.Exit(1)
		}

		// Execute the command using shell
		execCmd := exec.Command("sh", "-c", commandStr)
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		// Run the command and capture exit code
		err := execCmd.Run()
		var exitCode int
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				} else {
					exitCode = 1
				}
			} else {
				// Command failed to start or other error
				fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
				exitCode = 1
			}
		} else {
			exitCode = 0
		}

		// Always run sprint after command completes (regardless of exit code)
		fmt.Println("\n--- Running sprint session ---")
		if err := runSprintProgrammatically(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running sprint: %v\n", err)
			// Don't override the original command's exit code
		}

		// Exit with the original command's exit code
		os.Exit(exitCode)
	},
}

func init() {
	rootCmd.AddCommand(waitCmd)
}
