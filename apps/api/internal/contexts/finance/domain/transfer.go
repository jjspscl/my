package domain

import (
	"fmt"
	"time"
)

type WalletTransfer struct {
	ID              string
	UserEmail       string
	FromWalletID    string
	ToWalletID      string
	FromAmountCents int64
	ToAmountCents   int64
	Description     string
	TransferDate    time.Time
	CreatedAt       time.Time
	IdempotencyKey  string
}

// NewWalletTransfer validates and creates a WalletTransfer. FromAmountCents is
// the amount leaving the source wallet (in the source wallet's currency) and
// ToAmountCents is the amount arriving in the destination wallet (in the
// destination wallet's currency). For same-currency transfers the two are
// equal; for cross-currency transfers the ratio captures the effective rate at
// transfer time.
func NewWalletTransfer(id, userEmail, fromWalletID, toWalletID, description string, fromAmountCents, toAmountCents int64, transferDate time.Time) (*WalletTransfer, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if fromWalletID == "" {
		return nil, fmt.Errorf("from wallet is required")
	}
	if toWalletID == "" {
		return nil, fmt.Errorf("to wallet is required")
	}
	if fromWalletID == toWalletID {
		return nil, fmt.Errorf("cannot transfer to same wallet")
	}
	if fromAmountCents <= 0 {
		return nil, fmt.Errorf("from amount must be positive")
	}
	if toAmountCents <= 0 {
		return nil, fmt.Errorf("to amount must be positive")
	}
	if transferDate.IsZero() {
		return nil, fmt.Errorf("transfer date is required")
	}

	return &WalletTransfer{
		ID:              id,
		UserEmail:       userEmail,
		FromWalletID:    fromWalletID,
		ToWalletID:      toWalletID,
		FromAmountCents: fromAmountCents,
		ToAmountCents:   toAmountCents,
		Description:     description,
		TransferDate:    transferDate,
		CreatedAt:       time.Now().UTC(),
	}, nil
}
