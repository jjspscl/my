package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

// TxCoordinator runs a function inside a single database transaction. The
// infrastructure Coordinator satisfies it; tests may substitute a fake.
type TxCoordinator interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// errIdempotentReplay aborts a transaction when a concurrent insert won the
// idempotency unique-index race; the caller then re-reads the winning row.
var errIdempotentReplay = errors.New("idempotent replay")

type GoalService struct {
	goalRepo     domain.GoalRepository
	transferRepo domain.TransferRepository
	walletRepo   domain.WalletRepository
	coordinator  TxCoordinator
	clock        *timeutil.Clock
}

func NewGoalService(goalRepo domain.GoalRepository, transferRepo domain.TransferRepository, walletRepo domain.WalletRepository) *GoalService {
	return &GoalService{goalRepo: goalRepo, transferRepo: transferRepo, walletRepo: walletRepo, clock: timeutil.New(time.UTC)}
}

// NewGoalServiceNoTransfer creates a GoalService without transfer support.
func NewGoalServiceNoTransfer(goalRepo domain.GoalRepository, walletRepo domain.WalletRepository) *GoalService {
	return &GoalService{goalRepo: goalRepo, walletRepo: walletRepo, clock: timeutil.New(time.UTC)}
}

// WithClock pins the calendar used for goal summaries.
func (s *GoalService) WithClock(c *timeutil.Clock) *GoalService {
	s.clock = c
	return s
}

// WithCoordinator makes contribution + backing transfer writes atomic.
func (s *GoalService) WithCoordinator(c TxCoordinator) *GoalService {
	s.coordinator = c
	return s
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
	// FromAmountCents is the amount leaving the source wallet in the source
	// wallet's currency. It is required only when the source wallet currency
	// differs from the goal currency; otherwise both legs equal AmountCents.
	FromAmountCents *int64
	IdempotencyKey  string
}

func (s *GoalService) Create(ctx context.Context, userEmail string, input CreateGoalInput) (*domain.SavingsGoal, error) {
	wallet, err := ensureUsableWallet(ctx, s.walletRepo, userEmail, input.TargetWalletID)
	if err != nil {
		return nil, err
	}
	// Wallet is the currency authority: a goal is denominated in the currency
	// of its target wallet.
	goal, err := domain.NewSavingsGoal(uuid.New().String(), userEmail, input.Name, input.TargetAmountCents, input.TargetDate, input.TargetWalletID, wallet.Currency)
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
	wallet, err := ensureUsableWallet(ctx, s.walletRepo, userEmail, input.TargetWalletID)
	if err != nil {
		return nil, err
	}

	goal, err := domain.NewSavingsGoal(input.ID, userEmail, input.Name, input.TargetAmountCents, input.TargetDate, input.TargetWalletID, wallet.Currency)
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

	ids := make([]string, 0, len(goals))
	for _, g := range goals {
		ids = append(ids, g.ID)
	}

	// One batched query instead of one SUM per goal.
	currents, err := s.goalRepo.GetCurrentAmountsByGoals(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get current amounts: %w", err)
	}

	now := s.clock.Now()
	summaries := make([]domain.GoalSummary, 0, len(goals))
	for _, goal := range goals {
		summaries = append(summaries, domain.ComputeGoalSummary(goal, currents[goal.ID], now))
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

	if input.IdempotencyKey != "" {
		if len(input.IdempotencyKey) > domain.MaxIdempotencyLen {
			return nil, fmt.Errorf("idempotency key too long (max %d)", domain.MaxIdempotencyLen)
		}
		existing, err := s.goalRepo.FindContributionByIdempotencyKey(ctx, input.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	// Resolve the source wallet once so the transfer legs are correct.
	var sourceWallet *domain.Wallet
	if input.SourceWalletID != nil && *input.SourceWalletID != "" {
		sourceWallet, err = ensureUsableWallet(ctx, s.walletRepo, userEmail, *input.SourceWalletID)
		if err != nil {
			return nil, err
		}
	}

	// Contribution plus its backing transfer must be atomic in production.
	// When no coordinator is wired (unit tests), the writes still happen but
	// without the transaction wrapper.
	if sourceWallet != nil && goal.TargetWalletID != "" && s.transferRepo != nil {
		return s.addContributionWithTransfer(ctx, userEmail, goal, sourceWallet, input)
	}

	contribution, err := domain.NewGoalContribution(uuid.New().String(), input.GoalID, input.AmountCents, input.ContributedAt, input.Note, input.SourceWalletID, nil)
	if err != nil {
		return nil, err
	}
	contribution.IdempotencyKey = input.IdempotencyKey

	if err := s.goalRepo.SaveContribution(ctx, contribution); err != nil {
		if input.IdempotencyKey != "" && isUniqueViolation(err) {
			if existing, ferr := s.goalRepo.FindContributionByIdempotencyKey(ctx, input.IdempotencyKey); ferr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("save contribution: %w", err)
	}

	return contribution, nil
}

func (s *GoalService) addContributionWithTransfer(ctx context.Context, userEmail string, goal *domain.SavingsGoal, sourceWallet *domain.Wallet, input AddContributionInput) (*domain.GoalContribution, error) {
	// Dual-leg amounts: the goal leg is in the goal's currency; the source leg
	// is in the source wallet's currency. Same-currency wallets use AmountCents
	// for both; cross-currency requires the caller to supply the source leg.
	fromAmount := input.AmountCents
	if sourceWallet.Currency != goal.Currency {
		if input.FromAmountCents == nil {
			return nil, fmt.Errorf("source wallet currency %s differs from goal currency %s: fromAmountCents is required", sourceWallet.Currency, goal.Currency)
		}
		fromAmount = *input.FromAmountCents
	}

	var contribution *domain.GoalContribution
	run := func(txCtx context.Context) error {
		transfer, err := domain.NewWalletTransfer(
			uuid.New().String(),
			userEmail,
			sourceWallet.ID,
			goal.TargetWalletID,
			fmt.Sprintf("Goal contribution: %s", goal.Name),
			fromAmount,
			input.AmountCents,
			input.ContributedAt,
		)
		if err != nil {
			return fmt.Errorf("create transfer: %w", err)
		}
		transfer.IdempotencyKey = input.IdempotencyKey
		if err := s.transferRepo.Save(txCtx, transfer); err != nil {
			return fmt.Errorf("save transfer: %w", err)
		}

		c, err := domain.NewGoalContribution(uuid.New().String(), input.GoalID, input.AmountCents, input.ContributedAt, input.Note, input.SourceWalletID, &transfer.ID)
		if err != nil {
			return err
		}
		c.IdempotencyKey = input.IdempotencyKey
		if err := s.goalRepo.SaveContribution(txCtx, c); err != nil {
			if input.IdempotencyKey != "" && isUniqueViolation(err) {
				return errIdempotentReplay
			}
			return fmt.Errorf("save contribution: %w", err)
		}
		contribution = c
		return nil
	}

	var err error
	if s.coordinator != nil {
		err = s.coordinator.WithTx(ctx, run)
	} else {
		err = run(ctx)
	}

	if errors.Is(err, errIdempotentReplay) {
		existing, ferr := s.goalRepo.FindContributionByIdempotencyKey(ctx, input.IdempotencyKey)
		if ferr == nil && existing != nil {
			return existing, nil
		}
		return nil, fmt.Errorf("save contribution: %w", err)
	}
	if err != nil {
		return nil, err
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

	summary := domain.ComputeGoalSummary(goal, current, s.clock.Now())
	return &summary, nil
}
