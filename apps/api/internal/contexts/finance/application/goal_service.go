package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type GoalService struct {
	goalRepo     domain.GoalRepository
	transferRepo domain.TransferRepository
}

func NewGoalService(goalRepo domain.GoalRepository, transferRepo domain.TransferRepository) *GoalService {
	return &GoalService{goalRepo: goalRepo, transferRepo: transferRepo}
}

// NewGoalServiceNoTransfer creates a GoalService without transfer support.
func NewGoalServiceNoTransfer(goalRepo domain.GoalRepository) *GoalService {
	return &GoalService{goalRepo: goalRepo}
}

type CreateGoalInput struct {
	Name              string
	TargetAmountCents int64
	TargetDate        *time.Time
	TargetWalletID    string
}

type UpdateGoalInput struct {
	ID                string
	Name              string
	TargetAmountCents int64
	TargetDate        *time.Time
	TargetWalletID    string
}

type AddContributionInput struct {
	GoalID         string
	AmountCents    int64
	ContributedAt  time.Time
	Note           *string
	SourceWalletID *string
}

func (s *GoalService) Create(ctx context.Context, userEmail string, input CreateGoalInput) (*domain.SavingsGoal, error) {
	goal, err := domain.NewSavingsGoal(uuid.New().String(), userEmail, input.Name, input.TargetAmountCents, input.TargetDate, input.TargetWalletID)
	if err != nil {
		return nil, err
	}

	if err := s.goalRepo.SaveGoal(ctx, goal); err != nil {
		return nil, fmt.Errorf("save goal: %w", err)
	}

	return goal, nil
}

func (s *GoalService) Update(ctx context.Context, userEmail string, input UpdateGoalInput) (*domain.SavingsGoal, error) {
	existing, err := s.goalRepo.FindGoalByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("find goal: %w", err)
	}
	if existing.UserEmail != userEmail {
		return nil, fmt.Errorf("goal not found")
	}

	goal, err := domain.NewSavingsGoal(input.ID, userEmail, input.Name, input.TargetAmountCents, input.TargetDate, input.TargetWalletID)
	if err != nil {
		return nil, err
	}
	goal.CreatedAt = existing.CreatedAt

	if err := s.goalRepo.UpdateGoal(ctx, goal); err != nil {
		return nil, fmt.Errorf("update goal: %w", err)
	}

	return goal, nil
}

func (s *GoalService) Delete(ctx context.Context, id, userEmail string) error {
	return s.goalRepo.DeleteGoal(ctx, id, userEmail)
}

func (s *GoalService) ListSummaries(ctx context.Context, userEmail string) ([]domain.GoalSummary, error) {
	goals, err := s.goalRepo.ListGoals(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list goals: %w", err)
	}

	summaries := make([]domain.GoalSummary, 0, len(goals))
	for _, goal := range goals {
		current, err := s.goalRepo.GetCurrentAmountByGoal(ctx, goal.ID)
		if err != nil {
			return nil, fmt.Errorf("get current amount: %w", err)
		}
		summaries = append(summaries, domain.ComputeGoalSummary(goal, current))
	}

	return summaries, nil
}

func (s *GoalService) AddContribution(ctx context.Context, userEmail string, input AddContributionInput) (*domain.GoalContribution, error) {
	goal, err := s.goalRepo.FindGoalByID(ctx, input.GoalID)
	if err != nil {
		return nil, fmt.Errorf("find goal: %w", err)
	}
	if goal.UserEmail != userEmail {
		return nil, fmt.Errorf("goal not found")
	}

	// If sourceWalletID is provided, auto-create a transfer to goal's target wallet
	var transferID *string
	if input.SourceWalletID != nil && goal.TargetWalletID != "" && s.transferRepo != nil {
		transfer, err := domain.NewWalletTransfer(
			uuid.New().String(),
			userEmail,
			*input.SourceWalletID,
			goal.TargetWalletID,
			fmt.Sprintf("Goal contribution: %s", goal.Name),
			input.AmountCents,
			input.ContributedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("create transfer: %w", err)
		}
		if err := s.transferRepo.Save(ctx, transfer); err != nil {
			return nil, fmt.Errorf("save transfer: %w", err)
		}
		transferID = &transfer.ID
	}

	contribution, err := domain.NewGoalContribution(uuid.New().String(), input.GoalID, input.AmountCents, input.ContributedAt, input.Note, input.SourceWalletID, transferID)
	if err != nil {
		return nil, err
	}

	if err := s.goalRepo.SaveContribution(ctx, contribution); err != nil {
		return nil, fmt.Errorf("save contribution: %w", err)
	}

	return contribution, nil
}

func (s *GoalService) GetGoalSummary(ctx context.Context, goalID, userEmail string) (*domain.GoalSummary, error) {
	goal, err := s.goalRepo.FindGoalByID(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("find goal: %w", err)
	}
	if goal.UserEmail != userEmail {
		return nil, fmt.Errorf("goal not found")
	}

	current, err := s.goalRepo.GetCurrentAmountByGoal(ctx, goalID)
	if err != nil {
		return nil, fmt.Errorf("get current amount: %w", err)
	}

	summary := domain.ComputeGoalSummary(goal, current)
	return &summary, nil
}
