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
	"srs/internal/domain"
	"srs/internal/scheduler"
	"srs/internal/store"
	"srs/internal/ui"

	"github.com/spf13/cobra"
)

var countFlag int
var minutesFlag int
var tagFlags []string
var excludeTagFlags []string

// SessionStats tracks statistics for a review session
type SessionStats struct {
	Reviewed int
	Again    int
	NextDue  *time.Time // nil if no cards due
}

// matchesTagFilters checks if a card matches the tag filters
func matchesTagFilters(card *domain.Card, includeTags []string, excludeTags []string) bool {
	// If include tags specified, card must have at least one
	if len(includeTags) > 0 {
		hasIncludeTag := false
		for _, includeTag := range includeTags {
			for _, cardTag := range card.Tags {
				if cardTag == includeTag {
					hasIncludeTag = true
					break
				}
			}
			if hasIncludeTag {
				break
			}
		}
		if !hasIncludeTag {
			return false
		}
	}

	// If exclude tags specified, card must not have any
	if len(excludeTags) > 0 {
		for _, excludeTag := range excludeTags {
			for _, cardTag := range card.Tags {
				if cardTag == excludeTag {
					return false
				}
			}
		}
	}

	return true
}

// runReviewSession executes a review session and returns session statistics
func runReviewSession(database *sql.DB, cardsPath string, countLimit int, minutesLimit int, includeTags []string, excludeTags []string) (*SessionStats, error) {
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

	// Filter cards by tags if needed
	var filteredStates []*domain.ReviewState
	for _, state := range dueStates {
		card, err := s.GetCard(state.CardID)
		if err != nil {
			// Skip cards we can't load
			continue
		}

		if matchesTagFilters(card, includeTags, excludeTags) {
			filteredStates = append(filteredStates, state)
		}
	}

	if len(filteredStates) == 0 {
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

	// Start time tracking
	startTime := time.Now()
	var timeLimit time.Duration
	if minutesLimit > 0 {
		timeLimit = time.Duration(minutesLimit) * time.Minute
	}

	// Create reader for input
	reader := bufio.NewReader(os.Stdin)

	// Get terminal width for rendering
	terminalWidth := ui.GetTerminalWidth()

	// Review loop
	for i, state := range filteredStates {
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
		cardHeader := fmt.Sprintf("--- Card %d/%d ---", i+1, len(filteredStates))
		fmt.Printf("\n%s\n\n", ui.ColorCardHeader(cardHeader))

		// Show question with wrapping
		renderedQuestion := ui.RenderCardContent(question, terminalWidth)
		fmt.Println(ui.ColorQuestion(renderedQuestion))
		fmt.Println(ui.ColorPrompt("\n[Press space or enter to reveal answer]"))

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

		// Show answer with wrapping
		renderedAnswer := ui.RenderCardContent(answer, terminalWidth)
		fmt.Println("\n" + ui.ColorAnswer(renderedAnswer))
		gradePrompt := fmt.Sprintf("\nGrade: %s=%s, %s=%s, %s=%s, %s=%s",
			ui.ColorGrade(1, "1"), ui.ColorGrade(1, "Again"),
			ui.ColorGrade(2, "2"), ui.ColorGrade(2, "Hard"),
			ui.ColorGrade(3, "3"), ui.ColorGrade(3, "Good"),
			ui.ColorGrade(4, "4"), ui.ColorGrade(4, "Easy"))
		fmt.Println(gradePrompt)

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
			fmt.Fprintf(os.Stderr, "Warning: failed to update review state: %v\n", err)
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
	fmt.Printf("\n%s\n", ui.ColorSummary("--- Session Summary ---"))
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

		// Run review session
		stats, err := runReviewSession(database, cardsPath, countFlag, minutesFlag, tagFlags, excludeTagFlags)
		if err != nil {
			handleError(err, "review session failed")
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
	reviewCmd.Flags().StringArrayVar(&tagFlags, "tag", []string{}, "Include only cards with this tag (can be repeated)")
	reviewCmd.Flags().StringArrayVar(&excludeTagFlags, "exclude-tag", []string{}, "Exclude cards with this tag (can be repeated)")
}
