package application

import (
	"context"
	"fmt"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// ensureUsableWallet verifies that wallet exists, belongs to user, and is active.
func ensureUsableWallet(ctx context.Context, repo domain.WalletRepository, userEmail, walletID string) (*domain.Wallet, error) {
	wallet, err := repo.FindByID(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found: %s", walletID)
	}
	if wallet.UserEmail != userEmail {
		return nil, fmt.Errorf("wallet not found: %s", walletID)
	}
	if wallet.ArchivedAt != nil {
		return nil, fmt.Errorf("wallet is archived: %s", walletID)
	}
	return wallet, nil
}
