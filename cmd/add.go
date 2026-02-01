package cmd

import (
	"fmt"

	"github.com/amrikmalhans/srs/internal/domain"
	"github.com/amrikmalhans/srs/internal/store"

	"github.com/spf13/cobra"
)

var tagsFlag []string

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new card",
	Long:  `Create a new card file with template content in the cards directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get cards path
		cardsPath, err := getCardsPath()
		if err != nil {
			handleError(err, "failed to get cards path")
		}

		// Create new card with template content
		card := domain.NewCard("# Q\n\n# A\n\n", tagsFlag)

		// Create store and write card
		s := store.NewStore(cardsPath)
		if err := s.CreateCard(card); err != nil {
			handleError(err, "failed to create card")
		}

		// Print success message
		fmt.Printf("Created card: %s\n", card.ID.String())
		fmt.Printf("Path: %s/%s.md\n", cardsPath, card.ID.String())
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	// Add tags flag
	addCmd.Flags().StringSliceVar(&tagsFlag, "tags", []string{}, "Tags for the card (e.g., --tags rust,ownership)")
}
