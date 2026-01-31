package cmd

import (
	"fmt"
	"os"
	"sort"

	"srs/internal/config"
	"srs/internal/store"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List cards and statistics",
	Long:  `List all cards and display statistics including total count and tag counts.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to get cards path: %v\n", err)
			os.Exit(1)
		}

		// Scan all cards
		s := store.NewStore(cardsPath)
		cards, err := s.ScanCards()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to scan cards: %v\n", err)
			os.Exit(1)
		}

		// Print total count
		fmt.Printf("Total cards: %d\n\n", len(cards))

		// Aggregate tags
		tagCounts := make(map[string]int)
		for _, card := range cards {
			for _, tag := range card.Tags {
				tagCounts[tag]++
			}
		}

		// Print tag statistics
		if len(tagCounts) > 0 {
			fmt.Println("Tags:")
			// Sort tags for consistent output
			var tags []string
			for tag := range tagCounts {
				tags = append(tags, tag)
			}
			sort.Strings(tags)

			for _, tag := range tags {
				fmt.Printf("  %s: %d\n", tag, tagCounts[tag])
			}
		} else {
			fmt.Println("Tags: (none)")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
