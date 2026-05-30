package domain

import (
	"fmt"
	"strings"
	"time"
)

type WalletKind string

const (
	WalletCash    WalletKind = "cash"
	WalletBank    WalletKind = "bank"
	WalletEwallet WalletKind = "ewallet"
)

type Wallet struct {
	ID                  string
	UserEmail           string
	Name                string
	Kind                WalletKind
	Currency            string
	OpeningBalanceCents int64
	IsDefault           bool
	ArchivedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewWallet(id, userEmail, name string, kind WalletKind, currency string, openingBalanceCents int64, isDefault bool) (*Wallet, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if kind != WalletCash && kind != WalletBank && kind != WalletEwallet {
		return nil, fmt.Errorf("invalid wallet kind: %s", kind)
	}
	if openingBalanceCents < 0 {
		return nil, fmt.Errorf("opening balance cannot be negative")
	}
	if currency == "" {
		currency = "PHP"
	}

	return &Wallet{
		ID:                  id,
		UserEmail:           userEmail,
		Name:                strings.TrimSpace(name),
		Kind:                kind,
		Currency:            currency,
		OpeningBalanceCents: openingBalanceCents,
		IsDefault:           isDefault,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}, nil
}

type WalletBalance struct {
	Wallet                Wallet
	BalanceCents          int64
	IncomeCents           int64
	ExpenseCents          int64
	IncomingTransferCents int64
	OutgoingTransferCents int64
}
