package scheduler

import (
	"math"
	"time"

	"srs/internal/domain"
)

// Grade represents user's rating of a card review
type Grade int

const (
	GradeAgain Grade = 0 // Again - reset card
	GradeHard  Grade = 1 // Hard - slight penalty
	GradeGood  Grade = 2 // Good - normal progression
	GradeEasy  Grade = 3 // Easy - bonus
)

// UpdateReviewState applies SM-2 algorithm based on grade and returns a new ReviewState
// The returned state has LastReviewedAt set to current time and DueAt updated based on interval
func UpdateReviewState(state *domain.ReviewState, grade Grade) *domain.ReviewState {
	now := time.Now()
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
		newState.Ease = math.Max(1.3, state.Ease-0.2)
		newState.Lapses = state.Lapses + 1
		newState.DueAt = now // Show again immediately

	case GradeHard:
		// Slight penalty
		newState.Ease = math.Max(1.3, state.Ease-0.15)
		if state.Reps == 0 {
			newState.IntervalDays = 1
		} else {
			newState.IntervalDays = int(math.Max(1, float64(state.IntervalDays)*1.2))
		}
		newState.Reps = state.Reps + 1
		newState.DueAt = now.AddDate(0, 0, newState.IntervalDays)

	case GradeGood:
		// Normal progression
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
		// Bonus
		if state.Reps == 0 {
			newState.IntervalDays = 4
		} else {
			newState.IntervalDays = int(math.Round(float64(state.IntervalDays) * state.Ease * 1.3))
		}
		newState.Reps = state.Reps + 1
		newState.Ease = math.Min(2.5, state.Ease+0.15)
		newState.DueAt = now.AddDate(0, 0, newState.IntervalDays)
	}

	return newState
}
