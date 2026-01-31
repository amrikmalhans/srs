package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"srs/internal/domain"
)

// ScanCards scans the cards directory and returns all valid cards
func ScanCards(cardsPath string) ([]*domain.Card, error) {
	filePaths, err := ScanCardFiles(cardsPath)
	if err != nil {
		return nil, err
	}

	var cards []*domain.Card
	var errors []string

	for _, filePath := range filePaths {
		card, err := ParseCardFile(filePath)
		if err != nil {
			// Log error but continue scanning
			errors = append(errors, fmt.Sprintf("failed to parse %s: %v", filePath, err))
			continue
		}
		cards = append(cards, card)
	}

	// If we have errors but no cards, return an error
	if len(cards) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to parse any cards: %s", strings.Join(errors, "; "))
	}

	// If we have some cards but some errors, we still return the cards
	// (errors are logged but don't stop the operation)
	return cards, nil
}

// ScanCardFiles returns all .md file paths in the cards directory (recursive)
func ScanCardFiles(cardsPath string) ([]string, error) {
	var filePaths []string

	err := filepath.Walk(cardsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log but continue walking
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .md files
		if strings.HasSuffix(strings.ToLower(path), ".md") {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk cards directory: %w", err)
	}

	return filePaths, nil
}
