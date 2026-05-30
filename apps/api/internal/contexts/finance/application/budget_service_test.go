package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// --- Mock BudgetRepository ---

type mockBudgetRepo struct {
	budgets    map[string]*domain.Budget
	categories map[string][]*domain.BudgetCategory
	spentMap   map[string]int64
}

func newMockBudgetRepo() *mockBudgetRepo {
	return &mockBudgetRepo{
		budgets:    make(map[string]*domain.Budget),
		categories: make(map[string][]*domain.BudgetCategory),
		spentMap:   make(map[string]int64),
	}
}

func (m *mockBudgetRepo) UpsertBudget(_ context.Context, b *domain.Budget) error {
	m.budgets[b.UserEmail+":"+b.Month] = b
	return nil
}

func (m *mockBudgetRepo) FindBudgetByMonth(_ context.Context, userEmail, month string) (*domain.Budget, error) {
	return m.budgets[userEmail+":"+month], nil
}

func (m *mockBudgetRepo) UpsertBudgetCategories(_ context.Context, budgetID string, cats []*domain.BudgetCategory) error {
	m.categories[budgetID] = cats
	return nil
}

func (m *mockBudgetRepo) GetBudgetCategories(_ context.Context, budgetID string) ([]*domain.BudgetCategory, error) {
	return m.categories[budgetID], nil
}

func (m *mockBudgetRepo) GetSpentByCategory(_ context.Context, _, _ string) (map[string]int64, error) {
	return m.spentMap, nil
}

// --- Tests ---

func TestUpsertBudget_CreateNew(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	budget, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 500000},
			{Category: "Transport", AllocatedCents: 200000},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "2026-05", budget.Month)
	assert.Len(t, repo.categories[budget.ID], 2)
}

func TestUpsertBudget_UpdateExisting(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	// Create first version
	budget, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 300000},
		},
	})
	require.NoError(t, err)

	// Update
	_, err = svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 500000},
			{Category: "Transport", AllocatedCents: 200000},
		},
	})
	require.NoError(t, err)
	assert.Len(t, repo.categories[budget.ID], 2)
}

func TestUpsertBudget_InvalidMonth(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	_, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month:      "bad",
		Categories: []BudgetCategoryInput{},
	})
	assert.Error(t, err)
}

func TestUpsertBudget_NegativeAllocation(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	_, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: -100},
		},
	})
	assert.Error(t, err)
}

func TestGetSummary_NoBudget(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	summary, err := svc.GetSummary(context.Background(), "user@test.com", "2026-05")
	require.NoError(t, err)
	assert.Equal(t, "2026-05", summary.Month)
	assert.Empty(t, summary.Categories)
	assert.Equal(t, int64(0), summary.TotalAllocatedCents)
}

func TestGetSummary_WithSpending(t *testing.T) {
	repo := newMockBudgetRepo()
	repo.spentMap = map[string]int64{"Food": 300000, "Transport": 50000}
	svc := NewBudgetService(repo)

	_, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 500000},
			{Category: "Transport", AllocatedCents: 200000},
		},
	})
	require.NoError(t, err)

	summary, err := svc.GetSummary(context.Background(), "user@test.com", "2026-05")
	require.NoError(t, err)
	assert.Equal(t, int64(700000), summary.TotalAllocatedCents)
	assert.Equal(t, int64(350000), summary.TotalSpentCents)
	assert.Equal(t, int64(350000), summary.TotalRemainingCents)
	assert.Len(t, summary.Categories, 2)
}

func TestGetSummary_OverspentCategory(t *testing.T) {
	repo := newMockBudgetRepo()
	repo.spentMap = map[string]int64{"Food": 600000} // spent more than allocated 500000
	svc := NewBudgetService(repo)

	_, err := svc.UpsertBudget(context.Background(), "user@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 500000},
		},
	})
	require.NoError(t, err)

	summary, err := svc.GetSummary(context.Background(), "user@test.com", "2026-05")
	require.NoError(t, err)
	assert.Equal(t, int64(-100000), summary.TotalRemainingCents)
	assert.Equal(t, int64(-100000), summary.Categories[0].RemainingCents)
}

func TestGetSummary_MultipleUsers(t *testing.T) {
	repo := newMockBudgetRepo()
	svc := NewBudgetService(repo)

	_, err := svc.UpsertBudget(context.Background(), "a@test.com", UpsertBudgetInput{
		Month: "2026-05",
		Categories: []BudgetCategoryInput{
			{Category: "Food", AllocatedCents: 100000},
		},
	})
	require.NoError(t, err)

	// Different user should have no budget
	summary, err := svc.GetSummary(context.Background(), "b@test.com", "2026-05")
	require.NoError(t, err)
	assert.Empty(t, summary.Categories)
}
