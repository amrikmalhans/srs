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

// ReviewState represents the scheduling state of a card in the database
type ReviewState struct {
	ID           uuid.UUID
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReview   time.Time
	LastReview   *time.Time // Nullable
	CreatedAt    time.Time
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
	if rs.ID == uuid.Nil {
		return fmt.Errorf("review state ID cannot be nil")
	}
	if rs.EaseFactor < 1.3 {
		return fmt.Errorf("ease factor must be at least 1.3")
	}
	if rs.IntervalDays < 0 {
		return fmt.Errorf("interval days cannot be negative")
	}
	if rs.Repetitions < 0 {
		return fmt.Errorf("repetitions cannot be negative")
	}
	return nil
}

// IsDue checks if the card is due for review
func (rs *ReviewState) IsDue() bool {
	return time.Now().After(rs.NextReview) || time.Now().Equal(rs.NextReview)
}
