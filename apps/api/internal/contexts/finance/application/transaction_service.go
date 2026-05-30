package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// BillAutoMatcher is an optional hook to auto-match bills after a transaction is created.
type BillAutoMatcher interface {
	TryAutoMatch(ctx context.Context, tx *domain.Transaction)
}

type TransactionService struct {
	repo        domain.TransactionRepository
	walletRepo  domain.WalletRepository
	currency    string
	billMatcher BillAutoMatcher
}

func NewTransactionService(repo domain.TransactionRepository, walletRepo domain.WalletRepository, currency string) *TransactionService {
	return &TransactionService{repo: repo, walletRepo: walletRepo, currency: currency}
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
}

func (s *TransactionService) resolveWallet(ctx context.Context, userEmail, walletID string) (*domain.Wallet, error) {
	if walletID == "" {
		defaultWallet, err := s.walletRepo.FindDefault(ctx, userEmail)
		if err != nil {
			return nil, fmt.Errorf("no wallet specified and no default wallet found: %w", err)
		}
		return defaultWallet, nil
	}

	wallet, err := s.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %w", err)
	}
	if wallet.UserEmail != userEmail {
		return nil, fmt.Errorf("wallet not found")
	}
	if wallet.ArchivedAt != nil {
		return nil, fmt.Errorf("wallet is archived")
	}
	return wallet, nil
}

func (s *TransactionService) Create(ctx context.Context, userEmail string, input CreateTransactionInput) (*domain.Transaction, error) {
	wallet, err := s.resolveWallet(ctx, userEmail, input.WalletID)
	if err != nil {
		return nil, err
	}

	tx, err := domain.NewTransaction(
		uuid.New().String(),
		userEmail,
		s.currency,
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

	if err := s.repo.Save(ctx, tx); err != nil {
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

func (s *TransactionService) GetTodayTotal(ctx context.Context, userEmail string) (*domain.DailyTotal, error) {
	now := time.Now().UTC()
	total, err := s.repo.GetTodayTotal(ctx, userEmail, now)
	if err != nil {
		return nil, fmt.Errorf("get today total: %w", err)
	}

	if total == nil {
		total = &domain.DailyTotal{
			Date:     now.Format("2006-01-02"),
			Currency: s.currency,
		}
	}

	return total, nil
}

func (s *TransactionService) Delete(ctx context.Context, id, userEmail string) error {
	if err := s.repo.Delete(ctx, id, userEmail); err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	return nil
}
