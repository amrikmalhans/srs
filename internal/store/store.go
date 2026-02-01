// Package store handles file system operations for card storage.
// It provides functions for reading, writing, parsing, and scanning markdown card files.
// Cards are stored as individual markdown files with YAML frontmatter containing
// metadata (ID, created timestamp, tags) and markdown content for questions and answers.
package store

import (
	"fmt"

	"github.com/amrikmalhans/srs/internal/domain"

	"github.com/google/uuid"
)

// Store handles card file operations
type Store struct {
	cardsPath string
}

// NewStore creates a new Store instance
func NewStore(cardsPath string) *Store {
	return &Store{
		cardsPath: cardsPath,
	}
}

// CreateCard writes a card to disk
func (s *Store) CreateCard(card *domain.Card) error {
	return WriteCard(s.cardsPath, card)
}

// ScanCards scans all cards from disk
func (s *Store) ScanCards() ([]*domain.Card, error) {
	return ScanCards(s.cardsPath)
}

// GetCard retrieves a card by ID
func (s *Store) GetCard(id uuid.UUID) (*domain.Card, error) {
	cards, err := s.ScanCards()
	if err != nil {
		return nil, err
	}

	for _, card := range cards {
		if card.ID == id {
			return card, nil
		}
	}

	return nil, fmt.Errorf("card not found: %s", id.String())
}
