package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"srs/internal/config"
	"srs/internal/db"
	"srs/internal/scheduler"
	"srs/internal/store"

	"github.com/spf13/cobra"
)

var countFlag int
var minutesFlag int

// SessionStats tracks statistics for a review session
type SessionStats struct {
	Reviewed int
	Again    int
	NextDue  *time.Time // nil if no cards due
}

// runReviewSession executes a review session and returns session statistics
func runReviewSession(database *sql.DB, cardsPath string, countLimit int, minutesLimit int) (*SessionStats, error) {
	// Fetch due cards
	limit := countLimit
	if limit <= 0 {
		limit = 0 // No limit
	}
	dueStates, err := db.GetDueCards(database, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due cards: %w", err)
	}

	stats := &SessionStats{}

	if len(dueStates) == 0 {
		// Check if there are any cards due for next due time
		nextDue, err := db.GetNextDueTime(database)
		if err != nil {
			// Non-fatal error, continue
			stats.NextDue = nil
		} else {
			stats.NextDue = nextDue
		}
		return stats, nil
	}

	// Create store for loading cards
	s := store.NewStore(cardsPath)

	// Start time tracking
	startTime := time.Now()
	var timeLimit time.Duration
	if minutesLimit > 0 {
		timeLimit = time.Duration(minutesLimit) * time.Minute
	}

	// Create reader for input
	reader := bufio.NewReader(os.Stdin)

	// Review loop
	for i, state := range dueStates {
		// Check time limit
		if minutesLimit > 0 {
			elapsed := time.Since(startTime)
			if elapsed >= timeLimit {
				fmt.Printf("\nTime limit reached (%d minutes).\n", minutesLimit)
				break
			}
		}

		// Load card
		card, err := s.GetCard(state.CardID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load card %s: %v\n", state.CardID.String(), err)
			continue
		}

		// Extract Q and A
		question := store.ExtractQuestion(card.Content)
		answer := store.ExtractAnswer(card.Content)

		if question == "" || answer == "" {
			fmt.Fprintf(os.Stderr, "Warning: card %s has invalid Q/A format, skipping\n", state.CardID.String())
			continue
		}

		// Show progress
		fmt.Printf("\n--- Card %d/%d ---\n\n", i+1, len(dueStates))

		// Show question
		fmt.Println(question)
		fmt.Println("\n[Press space or enter to reveal answer]")

		// Wait for space or enter
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("error reading input: %w", err)
			}
			input = strings.TrimSpace(input)
			if input == "" || input == " " {
				break
			}
		}

		// Show answer
		fmt.Println("\n" + answer)
		fmt.Println("\nGrade: 1=Again, 2=Hard, 3=Good, 4=Easy")

		// Wait for grade
		var grade scheduler.Grade
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("error reading input: %w", err)
			}
			input = strings.TrimSpace(input)

			gradeNum, err := strconv.Atoi(input)
			if err != nil || gradeNum < 1 || gradeNum > 4 {
				fmt.Print("Invalid input. Please enter 1, 2, 3, or 4: ")
				continue
			}

			// Convert 1-4 to 0-3 (Again/Hard/Good/Easy)
			grade = scheduler.Grade(gradeNum - 1)
			break
		}

		// Update review state
		clock := scheduler.NewRealClock()
		updatedState := scheduler.UpdateReviewState(clock, state, grade)
		if err := db.UpsertReviewState(database, updatedState); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to update review state: %v\n", err)
			continue
		}

		stats.Reviewed++

		// Track "Again" grades (Grade 0)
		if grade == scheduler.GradeAgain {
			stats.Again++
		}
	}

	// Get next due time after session
	nextDue, err := db.GetNextDueTime(database)
	if err != nil {
		// Non-fatal error, continue without next due time
		stats.NextDue = nil
	} else {
		stats.NextDue = nextDue
	}

	return stats, nil
}

// displaySessionSummary displays the session summary
func displaySessionSummary(stats *SessionStats) {
	fmt.Printf("\n--- Session Summary ---\n")
	fmt.Printf("Reviewed: %d\n", stats.Reviewed)
	fmt.Printf("Again: %d\n", stats.Again)
	if stats.NextDue != nil {
		fmt.Printf("Next due: %s\n", stats.NextDue.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Next due: No cards due\n")
	}
}

// reviewCmd represents the review command
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Start a review session",
	Long:  `Start a review session showing due cards. Press space to reveal answer, then grade with 1-4.`,
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

		// Run review session
		stats, err := runReviewSession(database, cardsPath, countFlag, minutesFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Display summary if any cards were reviewed
		if stats.Reviewed > 0 {
			displaySessionSummary(stats)
		} else {
			fmt.Println("No cards due for review.")
		}
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	// Add flags
	reviewCmd.Flags().IntVar(&countFlag, "count", 0, "Maximum number of cards to review (0 = unlimited)")
	reviewCmd.Flags().IntVar(&minutesFlag, "minutes", 0, "Maximum time in minutes (0 = unlimited)")
}
