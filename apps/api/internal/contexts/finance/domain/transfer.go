package domain

import (
	"fmt"
	"time"
)

type WalletTransfer struct {
	ID           string
	UserEmail    string
	FromWalletID string
	ToWalletID   string
	AmountCents  int64
	Description  string
	TransferDate time.Time
	CreatedAt    time.Time
}

func NewWalletTransfer(id, userEmail, fromWalletID, toWalletID, description string, amountCents int64, transferDate time.Time) (*WalletTransfer, error) {
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
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	if transferDate.IsZero() {
		return nil, fmt.Errorf("transfer date is required")
	}

	return &WalletTransfer{
		ID:           id,
		UserEmail:    userEmail,
		FromWalletID: fromWalletID,
		ToWalletID:   toWalletID,
		AmountCents:  amountCents,
		Description:  description,
		TransferDate: transferDate,
		CreatedAt:    time.Now().UTC(),
	}, nil
}
