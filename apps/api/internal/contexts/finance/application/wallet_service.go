package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type WalletService struct {
	walletRepo domain.WalletRepository
}

func NewWalletService(walletRepo domain.WalletRepository) *WalletService {
	return &WalletService{walletRepo: walletRepo}
}

type CreateWalletInput struct {
	Name                string
	Kind                domain.WalletKind
	OpeningBalanceCents int64
}

type UpdateWalletInput struct {
	ID                  string
	Name                string
	Kind                domain.WalletKind
	OpeningBalanceCents int64
}

func (s *WalletService) Create(ctx context.Context, userEmail, currency string, input CreateWalletInput) (*domain.Wallet, error) {
	// Check if this is the first wallet (make it default)
	existing, err := s.walletRepo.ListByUser(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	isDefault := len(existing) == 0

	wallet, err := domain.NewWallet(
		uuid.New().String(),
		userEmail,
		input.Name,
		input.Kind,
		currency,
		input.OpeningBalanceCents,
		isDefault,
	)
	if err != nil {
		return nil, err
	}

	if err := s.walletRepo.Save(ctx, wallet); err != nil {
		return nil, fmt.Errorf("save wallet: %w", err)
	}

	return wallet, nil
}

func (s *WalletService) List(ctx context.Context, userEmail string) ([]*domain.Wallet, error) {
	return s.walletRepo.ListByUser(ctx, userEmail)
}

func (s *WalletService) ListWithBalances(ctx context.Context, userEmail string) ([]*domain.WalletBalance, error) {
	return s.walletRepo.GetBalancesByUser(ctx, userEmail)
}

func (s *WalletService) Update(ctx context.Context, userEmail string, input UpdateWalletInput) (*domain.Wallet, error) {
	existing, err := s.walletRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("find wallet: %w", err)
	}
	if existing.UserEmail != userEmail {
		return nil, fmt.Errorf("wallet not found")
	}

	wallet, err := domain.NewWallet(
		input.ID,
		userEmail,
		input.Name,
		input.Kind,
		existing.Currency,
		input.OpeningBalanceCents,
		existing.IsDefault,
	)
	if err != nil {
		return nil, err
	}
	wallet.CreatedAt = existing.CreatedAt

	if err := s.walletRepo.Update(ctx, wallet); err != nil {
		return nil, fmt.Errorf("update wallet: %w", err)
	}

	return wallet, nil
}

func (s *WalletService) Archive(ctx context.Context, id, userEmail string) error {
	wallet, err := s.walletRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find wallet: %w", err)
	}
	if wallet.UserEmail != userEmail {
		return fmt.Errorf("wallet not found")
	}

	return s.walletRepo.Archive(ctx, id, userEmail)
}
