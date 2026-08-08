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

type mockGoalTransferRepo struct {
	transfers []*domain.WalletTransfer
}

func (m *mockGoalTransferRepo) Save(_ context.Context, transfer *domain.WalletTransfer) error {
	m.transfers = append(m.transfers, transfer)
	return nil
}

func (m *mockGoalTransferRepo) FindByID(_ context.Context, id string) (*domain.WalletTransfer, error) {
	for _, transfer := range m.transfers {
		if transfer.ID == id {
			return transfer, nil
		}
	}
	return nil, fmt.Errorf("transfer not found")
}

func (m *mockGoalTransferRepo) ListByUser(_ context.Context, userEmail string, _, _ int) ([]*domain.WalletTransfer, error) {
	var result []*domain.WalletTransfer
	for _, transfer := range m.transfers {
		if transfer.UserEmail == userEmail {
			result = append(result, transfer)
		}
	}
	return result, nil
}

func newMockGoalRepo() *mockGoalRepo {
	return &mockGoalRepo{
		goals:         make(map[string]*domain.SavingsGoal),
		contributions: make(map[string][]*domain.GoalContribution),
	}
}

func newGoalWalletRepo() *mockWalletRepo {
	return &mockWalletRepo{wallets: []*domain.Wallet{
		{ID: "w-1", UserEmail: "user@test.com"},
		{ID: "w-2", UserEmail: "user@test.com"},
	}}
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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

	summaries, err := svc.ListSummaries(context.Background(), "user@test.com")
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func TestListSummaries_WithGoal(t *testing.T) {
	repo := newMockGoalRepo()
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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
	svc := NewGoalServiceNoTransfer(repo, newGoalWalletRepo())

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

func TestCreateGoalRejectsUnusableWallet(t *testing.T) {
	tests := []struct {
		name   string
		wallet *domain.Wallet
		want   string
	}{
		{name: "unknown", want: "wallet not found"},
		{name: "foreign", wallet: &domain.Wallet{ID: "foreign", UserEmail: "other@test.com"}, want: "wallet not found"},
		{name: "archived", wallet: archivedWallet("w-archived", "user@test.com"), want: "wallet is archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockGoalRepo()
			walletRepo := &mockWalletRepo{}
			if tt.wallet != nil {
				walletRepo.wallets = []*domain.Wallet{tt.wallet}
			}
			svc := NewGoalServiceNoTransfer(repo, walletRepo)

			_, err := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
				Name:              "Protected goal",
				TargetAmountCents: 1000,
				TargetWalletID:    walletIDForTest(tt.wallet, "missing"),
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			assert.Empty(t, repo.goals)
		})
	}
}

func TestAddContributionValidatesSourceAndCreatesTransfer(t *testing.T) {
	goalRepo := newMockGoalRepo()
	transferRepo := &mockGoalTransferRepo{}
	walletRepo := newGoalWalletRepo()
	svc := NewGoalService(goalRepo, transferRepo, walletRepo)

	goal, err := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Vacation",
		TargetAmountCents: 10000,
		TargetWalletID:    "w-2",
	})
	require.NoError(t, err)

	_, err = svc.AddContribution(context.Background(), "user@test.com", AddContributionInput{
		GoalID:         goal.ID,
		AmountCents:    1000,
		ContributedAt:  time.Now(),
		SourceWalletID: stringPtr("w-1"),
	})
	require.NoError(t, err)
	assert.Len(t, transferRepo.transfers, 1)
	assert.Len(t, goalRepo.contributions[goal.ID], 1)
}

func TestAddContributionRejectsUnusableSourceBeforeWrite(t *testing.T) {
	goalRepo := newMockGoalRepo()
	transferRepo := &mockGoalTransferRepo{}
	walletRepo := newGoalWalletRepo()
	svc := NewGoalService(goalRepo, transferRepo, walletRepo)
	goal, err := svc.Create(context.Background(), "user@test.com", CreateGoalInput{
		Name:              "Vacation",
		TargetAmountCents: 10000,
		TargetWalletID:    "w-2",
	})
	require.NoError(t, err)

	_, err = svc.AddContribution(context.Background(), "user@test.com", AddContributionInput{
		GoalID:         goal.ID,
		AmountCents:    1000,
		ContributedAt:  time.Now(),
		SourceWalletID: stringPtr("missing"),
	})
	require.Error(t, err)
	assert.Empty(t, transferRepo.transfers)
	assert.Empty(t, goalRepo.contributions[goal.ID])
}

func archivedWallet(id, email string) *domain.Wallet {
	archivedAt := time.Now()
	return &domain.Wallet{ID: id, UserEmail: email, ArchivedAt: &archivedAt}
}

func walletIDForTest(wallet *domain.Wallet, fallback string) string {
	if wallet == nil {
		return fallback
	}
	return wallet.ID
}

func stringPtr(value string) *string { return &value }
