package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"srs/internal/domain"

	"go.yaml.in/yaml/v3"
)

// WriteCard writes a card to a file in the cards directory
func WriteCard(cardsPath string, card *domain.Card) error {
	// Format card as markdown
	content, err := FormatCard(card)
	if err != nil {
		return fmt.Errorf("failed to format card: %w", err)
	}

	// Create filename from UUID
	filename := card.ID.String() + ".md"
	filePath := filepath.Join(cardsPath, filename)

	// Ensure directory exists
	if err := os.MkdirAll(cardsPath, 0755); err != nil {
		return fmt.Errorf("failed to create cards directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write card file: %w", err)
	}

	return nil
}

// FormatCard formats a card as markdown bytes
func FormatCard(card *domain.Card) ([]byte, error) {
	var buf strings.Builder

	// Write frontmatter
	buf.WriteString("---\n")

	// Build YAML frontmatter
	frontmatter := map[string]interface{}{
		"id":      card.ID.String(),
		"created": card.Created.Format(time.RFC3339),
		"tags":    card.Tags,
	}

	// If tags is empty, ensure it's an empty array in YAML
	if card.Tags == nil {
		frontmatter["tags"] = []string{}
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	buf.Write(yamlData)
	buf.WriteString("---\n")

	// Write content body
	buf.WriteString(card.Content)
	if !strings.HasSuffix(card.Content, "\n") {
		buf.WriteString("\n")
	}

	return []byte(buf.String()), nil
}
