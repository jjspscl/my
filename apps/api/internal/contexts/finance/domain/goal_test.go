package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSavingsGoal_Valid(t *testing.T) {
	g, err := NewSavingsGoal("g-1", "user@test.com", "Emergency Fund", 50000000, nil, "w-1")
	require.NoError(t, err)
	assert.Equal(t, "Emergency Fund", g.Name)
	assert.Equal(t, int64(50000000), g.TargetAmountCents)
	assert.Nil(t, g.TargetDate)
	assert.Equal(t, "w-1", g.TargetWalletID)
	assert.NotZero(t, g.CreatedAt)
}

func TestNewSavingsGoal_WithTargetDate(t *testing.T) {
	td := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	g, err := NewSavingsGoal("g-2", "user@test.com", "Vacation", 20000000, &td, "w-1")
	require.NoError(t, err)
	assert.NotNil(t, g.TargetDate)
	assert.Equal(t, 2027, g.TargetDate.Year())
}

func TestNewSavingsGoal_EmptyName(t *testing.T) {
	_, err := NewSavingsGoal("g-3", "user@test.com", "  ", 100000, nil, "w-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestNewSavingsGoal_ZeroTarget(t *testing.T) {
	_, err := NewSavingsGoal("g-4", "user@test.com", "Test", 0, nil, "w-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target amount must be positive")
}

func TestNewSavingsGoal_NegativeTarget(t *testing.T) {
	_, err := NewSavingsGoal("g-5", "user@test.com", "Test", -100, nil, "w-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target amount must be positive")
}

func TestNewSavingsGoal_NameTooLong(t *testing.T) {
	long := make([]byte, MaxGoalNameLen+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err := NewSavingsGoal("g-6", "user@test.com", string(long), 1000, nil, "w-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

func TestNewGoalContribution_Valid(t *testing.T) {
	c, err := NewGoalContribution("c-1", "g-1", 500000, time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), strPtr("First deposit"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(500000), c.AmountCents)
	assert.Equal(t, "First deposit", *c.Note)
	assert.NotZero(t, c.CreatedAt)
}

func TestNewGoalContribution_ZeroAmount(t *testing.T) {
	_, err := NewGoalContribution("c-2", "g-1", 0, time.Now(), nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestNewGoalContribution_NoNote(t *testing.T) {
	c, err := NewGoalContribution("c-3", "g-1", 1000, time.Now(), nil, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, c.Note)
}

func TestComputeGoalSummary_InProgress(t *testing.T) {
	g, _ := NewSavingsGoal("g-1", "u@t.com", "Goal", 100000, nil, "w-1")
	s := ComputeGoalSummary(g, 30000)
	assert.Equal(t, int64(30000), s.CurrentAmountCents)
	assert.Equal(t, int64(70000), s.RemainingAmountCents)
	assert.Equal(t, 30, s.ProgressPercent)
	assert.Equal(t, GoalInProgress, s.Status)
	assert.Nil(t, s.RequiredMonthlyCents)
}

func TestComputeGoalSummary_Achieved(t *testing.T) {
	g, _ := NewSavingsGoal("g-2", "u@t.com", "Goal", 100000, nil, "w-1")
	s := ComputeGoalSummary(g, 100000)
	assert.Equal(t, int64(0), s.RemainingAmountCents)
	assert.Equal(t, 100, s.ProgressPercent)
	assert.Equal(t, GoalAchieved, s.Status)
}

func TestComputeGoalSummary_OverAchieved(t *testing.T) {
	g, _ := NewSavingsGoal("g-3", "u@t.com", "Goal", 100000, nil, "w-1")
	s := ComputeGoalSummary(g, 150000)
	assert.Equal(t, int64(0), s.RemainingAmountCents)
	assert.Equal(t, 100, s.ProgressPercent)
	assert.Equal(t, GoalAchieved, s.Status)
}

func TestComputeGoalSummary_NotStarted(t *testing.T) {
	g, _ := NewSavingsGoal("g-4", "u@t.com", "Goal", 100000, nil, "w-1")
	s := ComputeGoalSummary(g, 0)
	assert.Equal(t, int64(0), s.CurrentAmountCents)
	assert.Equal(t, GoalNotStarted, s.Status)
	assert.Equal(t, 0, s.ProgressPercent)
}

func TestComputeGoalSummary_Behind(t *testing.T) {
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	g, _ := NewSavingsGoal("g-5", "u@t.com", "Goal", 100000, &past, "w-1")
	s := ComputeGoalSummary(g, 30000)
	assert.Equal(t, GoalBehind, s.Status)
}

func TestComputeGoalSummary_RequiredMonthly(t *testing.T) {
	future := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	g, _ := NewSavingsGoal("g-6", "u@t.com", "Goal", 120000, &future, "w-1")
	// Current saved: 0, target 120000, ~7 months from May 2026 to Dec 2026
	s := ComputeGoalSummary(g, 0)
	assert.NotNil(t, s.RequiredMonthlyCents)
	assert.True(t, *s.RequiredMonthlyCents > 0)
}

func TestComputeGoalSummary_AchievedNoMonthly(t *testing.T) {
	future := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	g, _ := NewSavingsGoal("g-7", "u@t.com", "Goal", 100000, &future, "w-1")
	s := ComputeGoalSummary(g, 100000)
	assert.Nil(t, s.RequiredMonthlyCents)
	assert.Equal(t, GoalAchieved, s.Status)
}

func TestMonthsBetween(t *testing.T) {
	from := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 11, monthsBetween(from, to))
}

func TestMonthsBetween_SameMonth(t *testing.T) {
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 1, monthsBetween(from, to))
}
