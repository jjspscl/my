package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// Used to detect idempotent replay races after a concurrent insert won the
// unique-index race.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// BillAutoMatcher is an optional hook to auto-match bills after a transaction is created.
type BillAutoMatcher interface {
	TryAutoMatch(ctx context.Context, tx *domain.Transaction)
}

type TransactionService struct {
	repo        domain.TransactionRepository
	walletRepo  domain.WalletRepository
	clock       *timeutil.Clock
	billMatcher BillAutoMatcher
}

func NewTransactionService(repo domain.TransactionRepository, walletRepo domain.WalletRepository, clock *timeutil.Clock) *TransactionService {
	return &TransactionService{repo: repo, walletRepo: walletRepo, clock: clock}
}

// WithBillAutoMatcher sets the bill auto-matcher hook.
func (s *TransactionService) WithBillAutoMatcher(m BillAutoMatcher) *TransactionService {
	s.billMatcher = m
	return s
}

type CreateTransactionInput struct {
	AmountCents     int64
	Category        string
	Description     string
	Type            domain.TransactionType
	WalletID        string
	TransactionDate time.Time
	IdempotencyKey  string
}

func (s *TransactionService) resolveWallet(ctx context.Context, userEmail, walletID string) (*domain.Wallet, error) {
	if walletID == "" {
		defaultWallet, err := s.walletRepo.FindDefault(ctx, userEmail)
		if err != nil {
			return nil, fmt.Errorf("no wallet specified and no default wallet found: %w", err)
		}
		return defaultWallet, nil
	}

	return ensureUsableWallet(ctx, s.walletRepo, userEmail, walletID)
}

func (s *TransactionService) Create(ctx context.Context, userEmail string, input CreateTransactionInput) (*domain.Transaction, error) {
	if input.IdempotencyKey != "" {
		if len(input.IdempotencyKey) > domain.MaxIdempotencyLen {
			return nil, fmt.Errorf("idempotency key too long (max %d)", domain.MaxIdempotencyLen)
		}
		existing, err := s.repo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	wallet, err := s.resolveWallet(ctx, userEmail, input.WalletID)
	if err != nil {
		return nil, err
	}

	// Wallet is the currency authority: a transaction is denominated in the
	// currency of the wallet it is booked against, never in a global default.
	tx, err := domain.NewTransaction(
		uuid.New().String(),
		userEmail,
		wallet.Currency,
		input.Category,
		input.Description,
		input.AmountCents,
		input.Type,
		input.TransactionDate,
	)
	if err != nil {
		return nil, err
	}
	tx.WalletID = wallet.ID
	tx.IdempotencyKey = input.IdempotencyKey

	if err := s.repo.Save(ctx, tx); err != nil {
		if input.IdempotencyKey != "" && isUniqueViolation(err) {
			// A concurrent replay won the unique-index race; return its row.
			if existing, ferr := s.repo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey); ferr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	if s.billMatcher != nil {
		s.billMatcher.TryAutoMatch(ctx, tx)
	}

	return tx, nil
}

type TransactionFilter struct {
	From   time.Time
	To     time.Time
	Limit  int
	Offset int
}

func (s *TransactionService) List(ctx context.Context, userEmail string, filter TransactionFilter) ([]*domain.Transaction, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	txs, err := s.repo.ListByUserAndDateRange(ctx, userEmail, filter.From, filter.To, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	return txs, nil
}

// TodayTotals returns today's income/expense/net totals grouped by currency,
// resolved against the user's financial calendar, never the server's UTC clock.
// When the day has no transactions the slice is empty; callers render zeros.
func (s *TransactionService) TodayTotals(ctx context.Context, userEmail string) ([]domain.CurrencyTotal, error) {
	totals, err := s.repo.GetTodayTotals(ctx, userEmail, s.clock.TodayStart())
	if err != nil {
		return nil, fmt.Errorf("get today totals: %w", err)
	}
	return totals, nil
}

// GetTodayTotal returns the single-currency daily total. For backwards
// compatibility it prefers the default currency and falls back to the first
// currency present. New code should use TodayTotals for per-currency results.
func (s *TransactionService) GetTodayTotal(ctx context.Context, userEmail, defaultCurrency string) (*domain.DailyTotal, error) {
	totals, err := s.TodayTotals(ctx, userEmail)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	total := &domain.DailyTotal{
		Date:     now.Format("2006-01-02"),
		Currency: defaultCurrency,
	}

	for _, t := range totals {
		if t.Currency == defaultCurrency {
			total.TotalCents = t.TotalCents
			total.ExpenseCents = t.ExpenseCents
			total.IncomeCents = t.IncomeCents
			return total, nil
		}
	}

	if len(totals) > 0 {
		total.Currency = totals[0].Currency
		total.TotalCents = totals[0].TotalCents
		total.ExpenseCents = totals[0].ExpenseCents
		total.IncomeCents = totals[0].IncomeCents
	}

	return total, nil
}

func (s *TransactionService) Delete(ctx context.Context, id, userEmail string) error {
	if err := s.repo.Delete(ctx, id, userEmail); err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	return nil
}
