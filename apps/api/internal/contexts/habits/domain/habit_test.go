package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHabit_Valid(t *testing.T) {
	h, err := NewHabit("h-001", "user@test.com", "Exercise", "green", FrequencyDaily, 7)
	require.NoError(t, err)
	assert.Equal(t, "Exercise", h.Name)
	assert.Equal(t, "green", h.Color)
	assert.Equal(t, FrequencyDaily, h.Frequency)
	assert.Equal(t, 7, h.TargetPerWeek)
	assert.False(t, h.Archived)
}

func TestNewHabit_Weekly(t *testing.T) {
	h, err := NewHabit("h-002", "user@test.com", "Learn Go", "indigo", FrequencyWeekly, 3)
	require.NoError(t, err)
	assert.Equal(t, FrequencyWeekly, h.Frequency)
	assert.Equal(t, 3, h.TargetPerWeek)
}

func TestNewHabit_EmptyName_Error(t *testing.T) {
	_, err := NewHabit("h-003", "user@test.com", "", "blue", FrequencyDaily, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestNewHabit_EmptyColor_DefaultsToBlue(t *testing.T) {
	h, err := NewHabit("h-004", "user@test.com", "Test", "", FrequencyDaily, 1)
	require.NoError(t, err)
	assert.Equal(t, "blue", h.Color)
}

func TestNewHabit_InvalidColor_Error(t *testing.T) {
	_, err := NewHabit("h-005", "user@test.com", "Test", "neon-pink", FrequencyDaily, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid color")
}

func TestNewHabit_InvalidFrequency_DefaultsToDaily(t *testing.T) {
	h, err := NewHabit("h-006", "user@test.com", "Test", "blue", "monthly", 1)
	require.NoError(t, err)
	assert.Equal(t, FrequencyDaily, h.Frequency)
}

func TestNewHabit_ZeroTarget_DefaultsToOne(t *testing.T) {
	h, err := NewHabit("h-007", "user@test.com", "Test", "blue", FrequencyDaily, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, h.TargetPerWeek)
}

func TestNewHabit_TargetOverSeven_CapsToSeven(t *testing.T) {
	h, err := NewHabit("h-008", "user@test.com", "Test", "blue", FrequencyWeekly, 10)
	require.NoError(t, err)
	assert.Equal(t, 7, h.TargetPerWeek)
}

func TestFrequency_Constants(t *testing.T) {
	assert.Equal(t, Frequency("daily"), FrequencyDaily)
	assert.Equal(t, Frequency("weekly"), FrequencyWeekly)
}

func TestHabitWithStatus_Embedding(t *testing.T) {
	h, _ := NewHabit("h-001", "user@test.com", "Exercise", "green", FrequencyDaily, 7)
	ws := HabitWithStatus{Habit: *h, CompletedToday: true, CurrentStreak: 5}
	assert.Equal(t, "h-001", ws.Habit.ID)
	assert.True(t, ws.CompletedToday)
	assert.Equal(t, 5, ws.CurrentStreak)
}

func TestHabitStreak_Fields(t *testing.T) {
	s := HabitStreak{
		HabitID: "h-001", Name: "Exercise", Color: "green",
		Frequency: FrequencyDaily, StreakCount: 10,
	}
	assert.Equal(t, 10, s.StreakCount)
}

func TestValidPaletteTokens_Has12(t *testing.T) {
	assert.Len(t, ValidPaletteTokens, 12)
}
