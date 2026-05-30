package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type TransferService struct {
	transferRepo domain.TransferRepository
	walletRepo   domain.WalletRepository
}

func NewTransferService(transferRepo domain.TransferRepository, walletRepo domain.WalletRepository) *TransferService {
	return &TransferService{transferRepo: transferRepo, walletRepo: walletRepo}
}

type CreateTransferInput struct {
	FromWalletID string
	ToWalletID   string
	AmountCents  int64
	Description  string
	TransferDate time.Time
}

func (s *TransferService) validateWallet(ctx context.Context, userEmail, walletID string) error {
	wallet, err := s.walletRepo.FindByID(ctx, walletID)
	if err != nil {
		return fmt.Errorf("wallet not found: %s", walletID)
	}
	if wallet.UserEmail != userEmail {
		return fmt.Errorf("wallet not found: %s", walletID)
	}
	if wallet.ArchivedAt != nil {
		return fmt.Errorf("wallet is archived: %s", walletID)
	}
	return nil
}

func (s *TransferService) Create(ctx context.Context, userEmail string, input CreateTransferInput) (*domain.WalletTransfer, error) {
	if err := s.validateWallet(ctx, userEmail, input.FromWalletID); err != nil {
		return nil, err
	}
	if err := s.validateWallet(ctx, userEmail, input.ToWalletID); err != nil {
		return nil, err
	}

	transfer, err := domain.NewWalletTransfer(
		uuid.New().String(),
		userEmail,
		input.FromWalletID,
		input.ToWalletID,
		input.Description,
		input.AmountCents,
		input.TransferDate,
	)
	if err != nil {
		return nil, err
	}

	if err := s.transferRepo.Save(ctx, transfer); err != nil {
		return nil, fmt.Errorf("save transfer: %w", err)
	}

	return transfer, nil
}

type TransferFilter struct {
	Limit  int
	Offset int
}

func (s *TransferService) List(ctx context.Context, userEmail string, filter TransferFilter) ([]*domain.WalletTransfer, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	transfers, err := s.transferRepo.ListByUser(ctx, userEmail, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}

	return transfers, nil
}
