package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type BudgetService struct {
	budgetRepo domain.BudgetRepository
}

func NewBudgetService(budgetRepo domain.BudgetRepository) *BudgetService {
	return &BudgetService{budgetRepo: budgetRepo}
}

type UpsertBudgetInput struct {
	Month      string
	Categories []BudgetCategoryInput
}

type BudgetCategoryInput struct {
	Category        string
	AllocatedCents  int64
	RolloverEnabled bool
}

func (s *BudgetService) UpsertBudget(ctx context.Context, userEmail string, input UpsertBudgetInput) (*domain.Budget, error) {
	budget, err := s.budgetRepo.FindBudgetByMonth(ctx, userEmail, input.Month)
	if err != nil {
		return nil, fmt.Errorf("find budget: %w", err)
	}

	if budget == nil {
		budget, err = domain.NewBudget(uuid.New().String(), userEmail, input.Month)
		if err != nil {
			return nil, err
		}
	}

	if err := s.budgetRepo.UpsertBudget(ctx, budget); err != nil {
		return nil, fmt.Errorf("upsert budget: %w", err)
	}

	categories := make([]*domain.BudgetCategory, 0, len(input.Categories))
	for _, c := range input.Categories {
		bc, err := domain.NewBudgetCategory(uuid.New().String(), budget.ID, c.Category, c.AllocatedCents, c.RolloverEnabled)
		if err != nil {
			return nil, fmt.Errorf("category %q: %w", c.Category, err)
		}
		categories = append(categories, bc)
	}

	if err := s.budgetRepo.UpsertBudgetCategories(ctx, budget.ID, categories); err != nil {
		return nil, fmt.Errorf("upsert categories: %w", err)
	}

	return budget, nil
}

func (s *BudgetService) GetSummary(ctx context.Context, userEmail, month string) (*domain.BudgetSummary, error) {
	budget, err := s.budgetRepo.FindBudgetByMonth(ctx, userEmail, month)
	if err != nil {
		return nil, fmt.Errorf("find budget: %w", err)
	}

	if budget == nil {
		return &domain.BudgetSummary{
			Month:      month,
			Categories: []domain.BudgetCategorySummary{},
		}, nil
	}

	categories, err := s.budgetRepo.GetBudgetCategories(ctx, budget.ID)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}

	spentMap, err := s.budgetRepo.GetSpentByCategory(ctx, userEmail, month)
	if err != nil {
		return nil, fmt.Errorf("get spent: %w", err)
	}

	summary := &domain.BudgetSummary{
		Month:      month,
		Categories: make([]domain.BudgetCategorySummary, 0, len(categories)),
	}

	for _, cat := range categories {
		spent := spentMap[cat.Category]
		remaining := cat.AllocatedCents - spent

		summary.Categories = append(summary.Categories, domain.BudgetCategorySummary{
			Category:        cat.Category,
			AllocatedCents:  cat.AllocatedCents,
			SpentCents:      spent,
			RemainingCents:  remaining,
			RolloverEnabled: cat.RolloverEnabled,
		})

		summary.TotalAllocatedCents += cat.AllocatedCents
		summary.TotalSpentCents += spent
	}
	summary.TotalRemainingCents = summary.TotalAllocatedCents - summary.TotalSpentCents

	return summary, nil
}
