// Package scheduler implements the SM-2 spaced repetition algorithm.
// It provides functions to update card review state based on user ratings,
// calculating next review dates and adjusting ease factors according to the
// SuperMemo 2 algorithm. The package uses a Clock interface for deterministic testing.
package scheduler

import (
	"math"
	"time"

	"github.com/amrikmalhans/srs/internal/domain"
)

// Grade represents user's rating of a card review
type Grade int

const (
	GradeAgain Grade = 0 // Again - reset card
	GradeHard  Grade = 1 // Hard - slight penalty
	GradeGood  Grade = 2 // Good - normal progression
	GradeEasy  Grade = 3 // Easy - bonus
)

// Ease factor constants
const (
	DefaultEase = 2.5 // Default ease factor for new cards
	MinEase     = 1.3 // Minimum ease factor (clamped)
)

// SameSessionThreshold defines the time window for same-session relearn
// If a card was reviewed within this duration, it's considered same-session
const SameSessionThreshold = 1 * time.Hour

// Clock interface for time abstraction to enable deterministic testing
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the system clock
type RealClock struct{}

// NewRealClock creates a new RealClock instance
func NewRealClock() Clock {
	return &RealClock{}
}

// Now returns the current system time
func (r *RealClock) Now() time.Time {
	return time.Now()
}

// FixedClock implements Clock with a fixed time for testing
type FixedClock struct {
	fixedTime time.Time
}

// NewFixedClock creates a new FixedClock with the given time
func NewFixedClock(t time.Time) Clock {
	return &FixedClock{fixedTime: t}
}

// Now returns the fixed time
func (f *FixedClock) Now() time.Time {
	return f.fixedTime
}

// UpdateReviewState applies SM-2 algorithm based on grade and returns a new ReviewState
// The returned state has LastReviewedAt set to current time and DueAt updated based on interval
// clock is used for deterministic time access (use NewRealClock() in production, NewFixedClock() in tests)
func UpdateReviewState(clock Clock, state *domain.ReviewState, grade Grade) *domain.ReviewState {
	now := clock.Now()
	newState := &domain.ReviewState{
		CardID:         state.CardID,
		IntervalDays:   state.IntervalDays,
		Ease:           state.Ease,
		Reps:           state.Reps,
		Lapses:         state.Lapses,
		LastReviewedAt: &now,
	}

	switch grade {
	case GradeAgain:
		// Reset card
		newState.Reps = 0
		newState.IntervalDays = 0
		newState.Ease = math.Max(MinEase, state.Ease-0.2)
		newState.Lapses = state.Lapses + 1

		// Same-session relearn: if reviewed within threshold, set due in 10 minutes
		// Otherwise, reset to immediate (0 days) or 1 day
		if state.LastReviewedAt != nil {
			timeSinceLastReview := now.Sub(*state.LastReviewedAt)
			if timeSinceLastReview <= SameSessionThreshold {
				// Same-session relearn: due in 10 minutes
				newState.DueAt = now.Add(10 * time.Minute)
			} else {
				// Not same-session: reset to 1 day
				newState.IntervalDays = 1
				newState.DueAt = now.AddDate(0, 0, 1)
			}
		} else {
			// First review failure: immediate
			newState.DueAt = now
		}

	case GradeHard:
		// Small interval growth, ease down slightly
		newState.Ease = math.Max(MinEase, state.Ease-0.15)
		if state.Reps == 0 {
			newState.IntervalDays = 1
		} else {
			newState.IntervalDays = int(math.Max(1, float64(state.IntervalDays)*1.2))
		}
		newState.Reps = state.Reps + 1
		newState.DueAt = now.AddDate(0, 0, newState.IntervalDays)

	case GradeGood:
		// Normal growth
		if state.Reps == 0 {
			newState.IntervalDays = 1
		} else if state.Reps == 1 {
			newState.IntervalDays = 6
		} else {
			newState.IntervalDays = int(math.Round(float64(state.IntervalDays) * state.Ease))
		}
		newState.Reps = state.Reps + 1
		newState.Ease = state.Ease // unchanged
		newState.DueAt = now.AddDate(0, 0, newState.IntervalDays)

	case GradeEasy:
		// Bigger growth, ease up slightly
		if state.Reps == 0 {
			newState.IntervalDays = 4
		} else {
			newState.IntervalDays = int(math.Round(float64(state.IntervalDays) * state.Ease * 1.3))
		}
		newState.Reps = state.Reps + 1
		newState.Ease = state.Ease + 0.15 // Increase ease (no max cap per requirements, but validate keeps it reasonable)
		newState.DueAt = now.AddDate(0, 0, newState.IntervalDays)
	}

	return newState
}
