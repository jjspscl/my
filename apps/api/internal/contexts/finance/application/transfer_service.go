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
	FromWalletID    string
	ToWalletID      string
	FromAmountCents int64
	ToAmountCents   int64
	Description     string
	TransferDate    time.Time
	IdempotencyKey  string
}

func (s *TransferService) Create(ctx context.Context, userEmail string, input CreateTransferInput) (*domain.WalletTransfer, error) {
	if input.IdempotencyKey != "" {
		if len(input.IdempotencyKey) > domain.MaxIdempotencyLen {
			return nil, fmt.Errorf("idempotency key too long (max %d)", domain.MaxIdempotencyLen)
		}
		existing, err := s.transferRepo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	if _, err := ensureUsableWallet(ctx, s.walletRepo, userEmail, input.FromWalletID); err != nil {
		return nil, err
	}
	if _, err := ensureUsableWallet(ctx, s.walletRepo, userEmail, input.ToWalletID); err != nil {
		return nil, err
	}

	transfer, err := domain.NewWalletTransfer(
		uuid.New().String(),
		userEmail,
		input.FromWalletID,
		input.ToWalletID,
		input.Description,
		input.FromAmountCents,
		input.ToAmountCents,
		input.TransferDate,
	)
	if err != nil {
		return nil, err
	}
	transfer.IdempotencyKey = input.IdempotencyKey

	if err := s.transferRepo.Save(ctx, transfer); err != nil {
		if input.IdempotencyKey != "" && isUniqueViolation(err) {
			if existing, ferr := s.transferRepo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey); ferr == nil && existing != nil {
				return existing, nil
			}
		}
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
