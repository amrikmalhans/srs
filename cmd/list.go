package cmd

import (
	"fmt"
	"sort"

	"srs/internal/config"
	"srs/internal/store"
	"srs/internal/ui"

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
			handleError(err, "failed to get cards path")
		}

		// Scan all cards
		s := store.NewStore(cardsPath)
		cards, err := s.ScanCards()
		if err != nil {
			handleError(err, "failed to scan cards")
		}

		// Print total count
		fmt.Printf("%s: %d\n\n", ui.ColorSummary("Total cards"), len(cards))

		// Aggregate tags
		tagCounts := make(map[string]int)
		for _, card := range cards {
			for _, tag := range card.Tags {
				tagCounts[tag]++
			}
		}

		// Print tag statistics
		if len(tagCounts) > 0 {
			fmt.Println(ui.ColorSummary("Tags:"))
			// Sort tags for consistent output
			var tags []string
			for tag := range tagCounts {
				tags = append(tags, tag)
			}
			sort.Strings(tags)

			for _, tag := range tags {
				fmt.Printf("  %s: %d\n", ui.ColorTag(tag), tagCounts[tag])
			}
		} else {
			fmt.Println(ui.ColorSummary("Tags: (none)"))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
