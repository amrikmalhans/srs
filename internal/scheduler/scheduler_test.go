package scheduler

import (
	"testing"
	"time"

	"srs/internal/domain"

	"github.com/google/uuid"
)

func TestUpdateReviewState_GoodIncreasesInterval(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name          string
		initialState  *domain.ReviewState
		expectedDays  int
		expectedReps  int
		expectedEase  float64
		expectedDueAt time.Time
	}{
		{
			name: "first review (reps=0) sets interval to 1 day",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   0,
				Ease:           2.5,
				Reps:           0,
				Lapses:         0,
				LastReviewedAt: nil,
			},
			expectedDays:  1,
			expectedReps:  1,
			expectedEase:  2.5,
			expectedDueAt: baseTime.AddDate(0, 0, 1),
		},
		{
			name: "second review (reps=1) sets interval to 6 days",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   1,
				Ease:           2.5,
				Reps:           1,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -1)),
			},
			expectedDays:  6,
			expectedReps:  2,
			expectedEase:  2.5,
			expectedDueAt: baseTime.AddDate(0, 0, 6),
		},
		{
			name: "subsequent review multiplies interval by ease",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   6,
				Ease:           2.5,
				Reps:           2,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -6)),
			},
			expectedDays:  15, // 6 * 2.5 = 15
			expectedReps:  3,
			expectedEase:  2.5,
			expectedDueAt: baseTime.AddDate(0, 0, 15),
		},
		{
			name: "mature card with higher ease factor",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   30,
				Ease:           2.8,
				Reps:           5,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -30)),
			},
			expectedDays:  84, // 30 * 2.8 = 84
			expectedReps:  6,
			expectedEase:  2.8,
			expectedDueAt: baseTime.AddDate(0, 0, 84),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, GradeGood)

			if result.IntervalDays != tt.expectedDays {
				t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, tt.expectedDays)
			}
			if result.Reps != tt.expectedReps {
				t.Errorf("Reps = %d, want %d", result.Reps, tt.expectedReps)
			}
			if result.Ease != tt.expectedEase {
				t.Errorf("Ease = %f, want %f", result.Ease, tt.expectedEase)
			}
			if !result.DueAt.Equal(tt.expectedDueAt) {
				t.Errorf("DueAt = %v, want %v", result.DueAt, tt.expectedDueAt)
			}
			if result.LastReviewedAt == nil {
				t.Error("LastReviewedAt should be set")
			} else if !result.LastReviewedAt.Equal(baseTime) {
				t.Errorf("LastReviewedAt = %v, want %v", *result.LastReviewedAt, baseTime)
			}
		})
	}
}

func TestUpdateReviewState_AgainResets(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name                string
		initialState        *domain.ReviewState
		expectedReps        int
		expectedInterval    int
		expectedLapses      int
		expectedEase        float64
		expectedDueAt       time.Time
		expectedSameSession bool
	}{
		{
			name: "first review failure - immediate reset",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   0,
				Ease:           2.5,
				Reps:           0,
				Lapses:         0,
				LastReviewedAt: nil,
			},
			expectedReps:        0,
			expectedInterval:    0,
			expectedLapses:      1,
			expectedEase:        2.3, // 2.5 - 0.2
			expectedDueAt:       baseTime,
			expectedSameSession: false,
		},
		{
			name: "same-session relearn - due in 10 minutes",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   1,
				Ease:           2.5,
				Reps:           1,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.Add(-30 * time.Minute)), // 30 minutes ago
			},
			expectedReps:        0,
			expectedInterval:    0,
			expectedLapses:      1,
			expectedEase:        2.3,
			expectedDueAt:       baseTime.Add(10 * time.Minute),
			expectedSameSession: true,
		},
		{
			name: "not same-session - reset to 1 day",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.5,
				Reps:           5,
				Lapses:         2,
				LastReviewedAt: timePtr(baseTime.Add(-2 * time.Hour)), // 2 hours ago (outside threshold)
			},
			expectedReps:        0,
			expectedInterval:    1,
			expectedLapses:      3,
			expectedEase:        2.3,
			expectedDueAt:       baseTime.AddDate(0, 0, 1),
			expectedSameSession: false,
		},
		{
			name: "exactly at threshold boundary - same session",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   5,
				Ease:           2.5,
				Reps:           3,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.Add(-1 * time.Hour)), // Exactly 1 hour ago
			},
			expectedReps:        0,
			expectedInterval:    0,
			expectedLapses:      1,
			expectedEase:        2.3,
			expectedDueAt:       baseTime.Add(10 * time.Minute),
			expectedSameSession: true,
		},
		{
			name: "just over threshold - not same session",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   5,
				Ease:           2.5,
				Reps:           3,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.Add(-1*time.Hour - 1*time.Second)), // Just over 1 hour
			},
			expectedReps:        0,
			expectedInterval:    1,
			expectedLapses:      1,
			expectedEase:        2.3,
			expectedDueAt:       baseTime.AddDate(0, 0, 1),
			expectedSameSession: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, GradeAgain)

			if result.Reps != tt.expectedReps {
				t.Errorf("Reps = %d, want %d", result.Reps, tt.expectedReps)
			}
			if result.IntervalDays != tt.expectedInterval {
				t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, tt.expectedInterval)
			}
			if result.Lapses != tt.expectedLapses {
				t.Errorf("Lapses = %d, want %d", result.Lapses, tt.expectedLapses)
			}
			if result.Ease != tt.expectedEase {
				t.Errorf("Ease = %f, want %f", result.Ease, tt.expectedEase)
			}
			if !result.DueAt.Equal(tt.expectedDueAt) {
				t.Errorf("DueAt = %v, want %v", result.DueAt, tt.expectedDueAt)
			}
			if result.LastReviewedAt == nil {
				t.Error("LastReviewedAt should be set")
			} else if !result.LastReviewedAt.Equal(baseTime) {
				t.Errorf("LastReviewedAt = %v, want %v", *result.LastReviewedAt, baseTime)
			}
		})
	}
}

func TestUpdateReviewState_EaseClamped(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name         string
		initialState *domain.ReviewState
		grade        Grade
		expectedEase float64
	}{
		{
			name: "Again rating clamps ease to minimum",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           1.4, // Close to minimum
				Reps:           5,
				Lapses:         2,
				LastReviewedAt: timePtr(baseTime.Add(-2 * time.Hour)),
			},
			grade:        GradeAgain,
			expectedEase: MinEase, // 1.4 - 0.2 = 1.2, clamped to 1.3
		},
		{
			name: "Again rating with ease at minimum stays at minimum",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           MinEase, // Already at minimum
				Reps:           5,
				Lapses:         2,
				LastReviewedAt: timePtr(baseTime.Add(-2 * time.Hour)),
			},
			grade:        GradeAgain,
			expectedEase: MinEase, // 1.3 - 0.2 = 1.1, clamped to 1.3
		},
		{
			name: "Hard rating clamps ease to minimum",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           1.4, // Close to minimum
				Reps:           5,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			grade:        GradeHard,
			expectedEase: MinEase, // 1.4 - 0.15 = 1.25, clamped to 1.3
		},
		{
			name: "Hard rating with ease at minimum stays at minimum",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           MinEase,
				Reps:           5,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			grade:        GradeHard,
			expectedEase: MinEase, // 1.3 - 0.15 = 1.15, clamped to 1.3
		},
		{
			name: "Easy rating increases ease without cap",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.0,
				Reps:           5,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			grade:        GradeEasy,
			expectedEase: 2.15, // 2.0 + 0.15
		},
		{
			name: "Easy rating can increase ease above 2.5",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.5,
				Reps:           5,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			grade:        GradeEasy,
			expectedEase: 2.65, // 2.5 + 0.15 (no cap per requirements)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, tt.grade)

			if result.Ease != tt.expectedEase {
				t.Errorf("Ease = %f, want %f", result.Ease, tt.expectedEase)
			}
			if result.Ease < MinEase {
				t.Errorf("Ease = %f, must be at least %f", result.Ease, MinEase)
			}
		})
	}
}

func TestUpdateReviewState_DeterministicDueDates(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	cardID := uuid.New()

	initialState := &domain.ReviewState{
		CardID:         cardID,
		DueAt:          baseTime,
		IntervalDays:   10,
		Ease:           2.5,
		Reps:           5,
		Lapses:         0,
		LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
	}

	// Test that same input produces same output
	clock1 := NewFixedClock(baseTime)
	result1 := UpdateReviewState(clock1, initialState, GradeGood)

	clock2 := NewFixedClock(baseTime)
	result2 := UpdateReviewState(clock2, initialState, GradeGood)

	if !result1.DueAt.Equal(result2.DueAt) {
		t.Errorf("DueAt not deterministic: first = %v, second = %v", result1.DueAt, result2.DueAt)
	}
	if result1.IntervalDays != result2.IntervalDays {
		t.Errorf("IntervalDays not deterministic: first = %d, second = %d", result1.IntervalDays, result2.IntervalDays)
	}
	if result1.Ease != result2.Ease {
		t.Errorf("Ease not deterministic: first = %f, second = %f", result1.Ease, result2.Ease)
	}

	// Test with different fixed times
	clock3 := NewFixedClock(baseTime.Add(1 * time.Hour))
	result3 := UpdateReviewState(clock3, initialState, GradeGood)

	// DueAt should be offset by the clock difference
	expectedDueAt3 := result1.DueAt.Add(1 * time.Hour)
	if !result3.DueAt.Equal(expectedDueAt3) {
		t.Errorf("DueAt not properly offset: got %v, want %v", result3.DueAt, expectedDueAt3)
	}
}

func TestUpdateReviewState_HardRating(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name          string
		initialState  *domain.ReviewState
		expectedDays  int
		expectedEase  float64
		expectedReps  int
		expectedDueAt time.Time
	}{
		{
			name: "Hard on first review sets interval to 1 day",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   0,
				Ease:           2.5,
				Reps:           0,
				Lapses:         0,
				LastReviewedAt: nil,
			},
			expectedDays:  1,
			expectedEase:  2.35, // 2.5 - 0.15
			expectedReps:  1,
			expectedDueAt: baseTime.AddDate(0, 0, 1),
		},
		{
			name: "Hard on subsequent review - small growth (1.2x)",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.5,
				Reps:           3,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			expectedDays:  12, // 10 * 1.2 = 12
			expectedEase:  2.35,
			expectedReps:  4,
			expectedDueAt: baseTime.AddDate(0, 0, 12),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, GradeHard)

			if result.IntervalDays != tt.expectedDays {
				t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, tt.expectedDays)
			}
			if result.Ease != tt.expectedEase {
				t.Errorf("Ease = %f, want %f", result.Ease, tt.expectedEase)
			}
			if result.Reps != tt.expectedReps {
				t.Errorf("Reps = %d, want %d", result.Reps, tt.expectedReps)
			}
			if !result.DueAt.Equal(tt.expectedDueAt) {
				t.Errorf("DueAt = %v, want %v", result.DueAt, tt.expectedDueAt)
			}
		})
	}
}

func TestUpdateReviewState_EasyRating(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name          string
		initialState  *domain.ReviewState
		expectedDays  int
		expectedEase  float64
		expectedReps  int
		expectedDueAt time.Time
	}{
		{
			name: "Easy on first review sets interval to 4 days",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   0,
				Ease:           2.5,
				Reps:           0,
				Lapses:         0,
				LastReviewedAt: nil,
			},
			expectedDays:  4,
			expectedEase:  2.65, // 2.5 + 0.15
			expectedReps:  1,
			expectedDueAt: baseTime.AddDate(0, 0, 4),
		},
		{
			name: "Easy on subsequent review - bigger growth (ease * 1.3)",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.5,
				Reps:           3,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			expectedDays:  33, // 10 * 2.5 * 1.3 = 32.5, rounded to 33
			expectedEase:  2.65,
			expectedReps:  4,
			expectedDueAt: baseTime.AddDate(0, 0, 33),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, GradeEasy)

			if result.IntervalDays != tt.expectedDays {
				t.Errorf("IntervalDays = %d, want %d", result.IntervalDays, tt.expectedDays)
			}
			if result.Ease != tt.expectedEase {
				t.Errorf("Ease = %f, want %f", result.Ease, tt.expectedEase)
			}
			if result.Reps != tt.expectedReps {
				t.Errorf("Reps = %d, want %d", result.Reps, tt.expectedReps)
			}
			if !result.DueAt.Equal(tt.expectedDueAt) {
				t.Errorf("DueAt = %v, want %v", result.DueAt, tt.expectedDueAt)
			}
		})
	}
}

func TestUpdateReviewState_EdgeCases(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	clock := NewFixedClock(baseTime)
	cardID := uuid.New()

	tests := []struct {
		name          string
		initialState  *domain.ReviewState
		grade         Grade
		expectedValid bool
	}{
		{
			name: "Hard with zero interval maintains minimum 1 day",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   0,
				Ease:           2.5,
				Reps:           0,
				Lapses:         0,
				LastReviewedAt: nil,
			},
			grade:         GradeHard,
			expectedValid: true,
		},
		{
			name: "Hard with very small interval (1 day) grows to 1 day",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   1,
				Ease:           2.5,
				Reps:           1,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -1)),
			},
			grade:         GradeHard,
			expectedValid: true,
		},
		{
			name: "Good preserves ease factor",
			initialState: &domain.ReviewState{
				CardID:         cardID,
				DueAt:          baseTime,
				IntervalDays:   10,
				Ease:           2.5,
				Reps:           3,
				Lapses:         0,
				LastReviewedAt: timePtr(baseTime.AddDate(0, 0, -10)),
			},
			grade:         GradeGood,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateReviewState(clock, tt.initialState, tt.grade)

			// Validate the result
			if err := result.Validate(); err != nil {
				if tt.expectedValid {
					t.Errorf("Result validation failed: %v", err)
				}
			} else if !tt.expectedValid {
				t.Error("Expected validation to fail but it passed")
			}

			// Ensure LastReviewedAt is always set
			if result.LastReviewedAt == nil {
				t.Error("LastReviewedAt should always be set")
			}

			// Ensure ease is never below minimum
			if result.Ease < MinEase {
				t.Errorf("Ease = %f, must be at least %f", result.Ease, MinEase)
			}
		})
	}
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
