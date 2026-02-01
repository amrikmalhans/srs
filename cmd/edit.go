package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/amrikmalhans/srs/internal/config"
	"github.com/amrikmalhans/srs/internal/db"
	"github.com/amrikmalhans/srs/internal/store"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// findCardByID finds a card file path by ID, supporting partial UUID matching
func findCardByID(cardsPath string, database *sql.DB, id string) (string, error) {
	// Try to parse as full UUID first
	if fullID, err := uuid.Parse(id); err == nil {
		// Try to get from database first
		if meta, err := db.GetCardMeta(database, fullID); err == nil && meta != nil {
			// Check if file exists
			if _, err := os.Stat(meta.Path); err == nil {
				return meta.Path, nil
			}
		}
		// Fallback: construct path
		filePath := filepath.Join(cardsPath, fullID.String()+".md")
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
		return "", fmt.Errorf("card not found: %s", id)
	}

	// Partial UUID matching - scan all cards
	cards, err := store.ScanCards(cardsPath)
	if err != nil {
		return "", fmt.Errorf("failed to scan cards: %w", err)
	}

	idLower := strings.ToLower(id)
	var matches []string

	for _, card := range cards {
		cardIDStr := strings.ToLower(card.ID.String())
		if strings.HasPrefix(cardIDStr, idLower) {
			filePath := filepath.Join(cardsPath, card.ID.String()+".md")
			matches = append(matches, filePath)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("card not found: %s", id)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple cards match %s: %d matches found", id, len(matches))
	}

	return matches[0], nil
}

// openInEditor opens a file in the user's $EDITOR
func openInEditor(filePath string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	// Parse editor command (handle cases like "code --wait")
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("invalid EDITOR environment variable")
	}

	cmd := exec.Command(parts[0], append(parts[1:], filePath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	return nil
}

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Open a card in your editor",
	Long:  `Open a card file in your $EDITOR. Supports full UUID or partial UUID matching (minimum 8 characters).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cardID := args[0]

		// Validate minimum length for partial UUID
		if len(cardID) < 8 {
			fmt.Fprintf(os.Stderr, "Error: card ID must be at least 8 characters\n")
			os.Exit(1)
		}

		// Get cards path
		cardsPath, err := config.CardsPath()
		if err != nil {
			handleError(err, "failed to get cards path")
		}

		// Open database
		database, err := openDatabase()
		if err != nil {
			handleError(err, "failed to open database")
		}
		defer db.CloseDB(database)

		// Find card file
		filePath, err := findCardByID(cardsPath, database, cardID)
		if err != nil {
			handleError(err, "failed to find card")
		}

		// Open in editor
		if err := openInEditor(filePath); err != nil {
			handleError(err, "failed to open editor")
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
