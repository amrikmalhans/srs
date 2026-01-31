// Package domain defines the core domain models for the SRS tool.
// It contains the Card, CardMeta, and ReviewState types that represent
// the business entities, along with validation and utility methods.
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Card represents a flashcard with frontmatter and content
type Card struct {
	ID      uuid.UUID
	Created time.Time
	Tags    []string
	Content string // Markdown content body
}

// CardMeta represents metadata about a card file
type CardMeta struct {
	CardID    uuid.UUID
	Path      string
	UpdatedAt time.Time
}

// ReviewState represents the scheduling state of a card in the database
type ReviewState struct {
	CardID         uuid.UUID
	DueAt          time.Time
	IntervalDays   int
	Ease           float64
	Reps           int
	Lapses         int
	LastReviewedAt *time.Time // Nullable
}

// NewCard creates a new card with generated UUID and current timestamp
func NewCard(content string, tags []string) *Card {
	return &Card{
		ID:      uuid.New(),
		Created: time.Now(),
		Tags:    tags,
		Content: content,
	}
}

// Validate checks if the card has required fields
func (c *Card) Validate() error {
	if c.ID == uuid.Nil {
		return fmt.Errorf("card ID cannot be nil")
	}
	if c.Content == "" {
		return fmt.Errorf("card content cannot be empty")
	}
	return nil
}

// ValidateReviewState checks if the review state is valid
func (rs *ReviewState) Validate() error {
	if rs.CardID == uuid.Nil {
		return fmt.Errorf("review state card ID cannot be nil")
	}
	if rs.Ease < 1.3 {
		return fmt.Errorf("ease factor must be at least 1.3")
	}
	if rs.IntervalDays < 0 {
		return fmt.Errorf("interval days cannot be negative")
	}
	if rs.Reps < 0 {
		return fmt.Errorf("reps cannot be negative")
	}
	if rs.Lapses < 0 {
		return fmt.Errorf("lapses cannot be negative")
	}
	return nil
}

// IsDue checks if the card is due for review
func (rs *ReviewState) IsDue() bool {
	return time.Now().After(rs.DueAt) || time.Now().Equal(rs.DueAt)
}
