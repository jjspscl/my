package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHabit_ValidCreation(t *testing.T) {
	h := &Habit{
		ID:            "h-001",
		UserEmail:     "user@test.com",
		Name:          "Exercise",
		Color:         "green",
		Frequency:     FrequencyDaily,
		TargetPerWeek: 7,
		Archived:      false,
		CreatedAt:     time.Now(),
	}

	assert.Equal(t, "Exercise", h.Name)
	assert.Equal(t, "green", h.Color)
	assert.Equal(t, FrequencyDaily, h.Frequency)
	assert.Equal(t, 7, h.TargetPerWeek)
}

func TestHabit_WeeklyFrequency(t *testing.T) {
	h := &Habit{
		Name:          "Learn Go",
		Frequency:     FrequencyWeekly,
		TargetPerWeek: 3,
	}

	assert.Equal(t, FrequencyWeekly, h.Frequency)
	assert.Equal(t, 3, h.TargetPerWeek)
}

func TestHabit_ArchivedByDefault(t *testing.T) {
	h := &Habit{
		Name:  "Test",
		Color: "blue",
	}

	assert.False(t, h.Archived)
}

func TestHabit_Defaults(t *testing.T) {
	h := &Habit{}

	assert.Empty(t, h.Name)
	assert.Empty(t, h.Color)
	assert.Empty(t, h.Frequency)
	assert.Equal(t, 0, h.TargetPerWeek)
}

func TestFrequency_Constants(t *testing.T) {
	assert.Equal(t, Frequency("daily"), FrequencyDaily)
	assert.Equal(t, Frequency("weekly"), FrequencyWeekly)
	assert.NotEqual(t, FrequencyDaily, FrequencyWeekly)
}

func TestHabitWithStatus_Embedding(t *testing.T) {
	h := HabitWithStatus{
		Habit: Habit{
			ID:    "h-001",
			Name:  "Exercise",
			Color: "green",
		},
		CompletedToday: true,
		CurrentStreak:  5,
	}

	assert.Equal(t, "h-001", h.Habit.ID)
	assert.True(t, h.CompletedToday)
	assert.Equal(t, 5, h.CurrentStreak)
}

func TestHabitCompletion_ValidCreation(t *testing.T) {
	c := &HabitCompletion{
		ID:            "hc-001",
		HabitID:       "h-001",
		CompletedDate: time.Now(),
		CreatedAt:     time.Now(),
	}
	assert.Equal(t, "h-001", c.HabitID)
	assert.False(t, c.CompletedDate.IsZero())
}

func TestHabitStreak_Fields(t *testing.T) {
	s := HabitStreak{
		HabitID:     "h-001",
		Name:        "Exercise",
		Color:       "green",
		Frequency:   FrequencyDaily,
		StreakCount: 10,
	}

	assert.Equal(t, "Exercise", s.Name)
	assert.Equal(t, 10, s.StreakCount)
}
