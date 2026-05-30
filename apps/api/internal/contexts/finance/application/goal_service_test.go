package application

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// --- Mock GoalRepository ---

type mockGoalRepo struct {
	goals         map[string]*domain.SavingsGoal
	contributions map[string][]*domain.GoalContribution // key: goalID
}

func newMockGoalRepo() *mockGoalRepo {
	return &mockGoalRepo{
		goals:         make(map[string]*domain.SavingsGoal),
		contributions: make(map[string][]*domain.GoalContribution),
	}
}

func (m *mockGoalRepo) SaveGoal(_ context.Context, goal *domain.SavingsGoal) error {
	m.goals[goal.ID] = goal
	return nil
}

func (m *mockGoalRepo) UpdateGoal(_ context.Context, goal *domain.SavingsGoal) error {
	m.goals[goal.ID] = goal
	return nil
}

func (m *mockGoalRepo) DeleteGoal(_ context.Context, id, userEmail string) error {
	g, ok := m.goals[id]
	if !ok || g.UserEmail != userEmail {
		return fmt.Errorf("goal not found")
	}
	delete(m.goals, id)
	return nil
}

func (m *mockGoalRepo) FindGoalByID(_ context.Context, id string) (*domain.SavingsGoal, error) {
	g, ok := m.goals[id]
	if !ok {
		return nil, fmt.Errorf("goal not found")
	}
	return g, nil
}

func (m *mockGoalRepo) ListGoals(_ context.Context, userEmail string) ([]*domain.SavingsGoal, error) {
	var result []*domain.SavingsGoal
	for _, g := range m.goals {
		if g.UserEmail == userEmail {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *mockGoalRepo) SaveContribution(_ context.Context, c *domain.GoalContribution) error {
	m.contributions[c.GoalID] = append(m.contributions[c.GoalID], c)
	return nil
}

func (m *mockGoalRepo) ListContributionsByGoal(_ context.Context, goalID string) ([]*domain.GoalContribution, error) {
	return m.contributions[goalID], nil
}

func (m *mockGoalRepo) GetCurrentAmountByGoal(_ context.Context, goalID string) (int64, error) {
	var total int64
	for _, c := range m.contributions[goalID] {
		total += c.AmountCents
	}
	return total, nil
}

// --- Tests ---

func TestCreateGoal_Valid(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, err := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Emergency Fund",
		TargetAmountCents: 50000000,
		TargetWalletID:    "w-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Emergency Fund", goal.Name)
	assert.Equal(t, int64(50000000), goal.TargetAmountCents)
	assert.NotEmpty(t, goal.ID)
}

func TestCreateGoal_InvalidTarget(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	_, err := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Test",
		TargetAmountCents: 0,
		TargetWalletID:    "w-1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target amount must be positive")
}

func TestListSummaries_Empty(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	summaries, err := svc.ListSummaries(context.Background(), "user@test.com")
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func TestListSummaries_WithGoal(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	_, _ = svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Goal 1",
		TargetAmountCents: 100000,
		TargetWalletID:    "w-1",
	})
	_, _ = svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Goal 2",
		TargetAmountCents: 200000,
		TargetWalletID:    "w-2",
	})

	summaries, err := svc.ListSummaries(context.Background(), "user@test.com")
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
}

func TestAddContribution_UpdatesSummary(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, _ := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Vacation",
		TargetAmountCents: 500000,
		TargetWalletID:    "w-1",
	})

	_, err := svc.AddContribution(context.Background(), "user@test.com", AddContributionInput{
		GoalID:        goal.ID,
		AmountCents:   100000,
		ContributedAt: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	summary, err := svc.GetGoalSummary(context.Background(), goal.ID, "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), summary.CurrentAmountCents)
	assert.Equal(t, int64(400000), summary.RemainingAmountCents)
	assert.Equal(t, 20, summary.ProgressPercent)
	assert.Equal(t, domain.GoalInProgress, summary.Status)
}

func TestAddContribution_AchievesGoal(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, _ := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Small Goal",
		TargetAmountCents: 1000,
		TargetWalletID:    "w-1",
	})

	_, err := svc.AddContribution(context.Background(), "user@test.com", AddContributionInput{
		GoalID:        goal.ID,
		AmountCents:   1000,
		ContributedAt: time.Now(),
	})
	require.NoError(t, err)

	summary, _ := svc.GetGoalSummary(context.Background(), goal.ID, "user@test.com")
	assert.Equal(t, domain.GoalAchieved, summary.Status)
	assert.Equal(t, 100, summary.ProgressPercent)
}

func TestAddContribution_WrongUser(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, _ := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Test",
		TargetAmountCents: 1000,
		TargetWalletID:    "w-1",
	})

	_, err := svc.AddContribution(context.Background(), "other@test.com", AddContributionInput{
		GoalID:        goal.ID,
		AmountCents:   500,
		ContributedAt: time.Now(),
	})
	assert.Error(t, err)
}

func TestDeleteGoal(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, _ := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Delete Me",
		TargetAmountCents: 10000,
		TargetWalletID:    "w-1",
	})

	err := svc.Delete(context.Background(), goal.ID, "user@test.com")
	require.NoError(t, err)

	summaries, _ := svc.ListSummaries(context.Background(), "user@test.com")
	assert.Empty(t, summaries)
}

func TestUpdateGoal(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo)

	goal, _ := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Original",
		TargetAmountCents: 50000,
		TargetWalletID:    "w-1",
	})

	updated, err := svc.Update(context.Background(), "user@test.com", UpdateGoalInput{
		ID:                goal.ID,
		Name:              "Updated",
		TargetAmountCents: 100000,
		TargetWalletID:    "w-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, int64(100000), updated.TargetAmountCents)
}
