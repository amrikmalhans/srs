package cmd

import (
	"fmt"
	"os"
	"strings"

	"srs/internal/config"
	"srs/internal/domain"
	"srs/internal/store"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// SearchResult represents a search match in a card
type SearchResult struct {
	CardID     uuid.UUID
	MatchField string // "Q", "A", or "tag"
	MatchText  string // The matching text snippet
	Tags       []string
}

// searchCards searches across all cards for the given query
func searchCards(cards []*domain.Card, query string) []*SearchResult {
	queryLower := strings.ToLower(query)
	var results []*SearchResult

	for _, card := range cards {
		// Search in tags
		for _, tag := range card.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, &SearchResult{
					CardID:     card.ID,
					MatchField: "tag",
					MatchText:  tag,
					Tags:       card.Tags,
				})
				break // Only add one result per card for tags
			}
		}

		// Search in question
		question := store.ExtractQuestion(card.Content)
		if question != "" && strings.Contains(strings.ToLower(question), queryLower) {
			results = append(results, &SearchResult{
				CardID:     card.ID,
				MatchField: "Q",
				MatchText:  truncateText(question, 80),
				Tags:       card.Tags,
			})
		}

		// Search in answer
		answer := store.ExtractAnswer(card.Content)
		if answer != "" && strings.Contains(strings.ToLower(answer), queryLower) {
			results = append(results, &SearchResult{
				CardID:     card.ID,
				MatchField: "A",
				MatchText:  truncateText(answer, 80),
				Tags:       card.Tags,
			})
		}
	}

	return results
}

// truncateText truncates text to maxLen characters, adding "..." if truncated
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	// Find word boundary if possible
	if maxLen > 3 {
		truncated := text[:maxLen-3]
		lastSpace := strings.LastIndex(truncated, " ")
		if lastSpace > maxLen/2 {
			return truncated[:lastSpace] + "..."
		}
	}
	return text[:maxLen-3] + "..."
}

// formatSearchResult formats a search result for display
func formatSearchResult(result *SearchResult) string {
	var sb strings.Builder

	// Card ID (shortened)
	cardIDStr := result.CardID.String()
	sb.WriteString(fmt.Sprintf("ID: %s", cardIDStr[:8]))
	sb.WriteString("\n")

	// Match field and text
	sb.WriteString(fmt.Sprintf("  Match in %s: %s", result.MatchField, result.MatchText))
	sb.WriteString("\n")

	// Tags
	if len(result.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("  Tags: %s", strings.Join(result.Tags, ", ")))
	} else {
		sb.WriteString("  Tags: (none)")
	}
	sb.WriteString("\n")

	return sb.String()
}

// findCmd represents the find command
var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search cards by text",
	Long:  `Search across card questions, answers, and tags. Case-insensitive matching.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		if strings.TrimSpace(query) == "" {
			fmt.Fprintf(os.Stderr, "Error: search query cannot be empty\n")
			os.Exit(1)
		}

		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			handleError(err, "failed to get cards path")
		}

		// Scan all cards
		s := store.NewStore(cardsPath)
		cards, err := s.ScanCards()
		if err != nil {
			handleError(err, "failed to scan cards")
		}

		// Search
		results := searchCards(cards, query)

		// Display results
		if len(results) == 0 {
			fmt.Printf("No cards found matching: %s\n", query)
			return
		}

		fmt.Printf("Found %d match(es) for: %s\n\n", len(results), query)
		for i, result := range results {
			if i > 0 {
				fmt.Println()
			}
			fmt.Print(formatSearchResult(result))
		}
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
}
