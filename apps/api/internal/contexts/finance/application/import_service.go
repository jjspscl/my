package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// ImportProviderCurrency maps a provider to the currency its statements are
// denominated in. The wallet is the currency authority, so an import may only
// be booked against a wallet of the provider's currency.
func ImportProviderCurrency(provider string) (string, error) {
	switch provider {
	case domain.ImportProviderGCashPDF:
		return "PHP", nil
	default:
		return "", fmt.Errorf("unsupported import provider: %s", provider)
	}
}

// ImportTxCoordinator runs a function inside one database transaction so the
// batch, entries, transactions, transfers, and wallet land atomically.
type ImportTxCoordinator interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type ImportService struct {
	importRepo   domain.ImportRepository
	txRepo       domain.TransactionRepository
	transferRepo domain.TransferRepository
	walletRepo   domain.WalletRepository
	coordinator  ImportTxCoordinator
}

func NewImportService(importRepo domain.ImportRepository, txRepo domain.TransactionRepository, transferRepo domain.TransferRepository, walletRepo domain.WalletRepository, coordinator ImportTxCoordinator) *ImportService {
	return &ImportService{
		importRepo:   importRepo,
		txRepo:       txRepo,
		transferRepo: transferRepo,
		walletRepo:   walletRepo,
		coordinator:  coordinator,
	}
}

type CreateWalletForImport struct {
	Name                string
	OpeningBalanceCents int64
}

type ImportRowInput struct {
	SourceReference string
	OccurredAt      time.Time
	AmountCents     int64
	Kind            string
	Category        string
	Description     string
	Counterparty    string
	CounterWalletID string
}

type CreateImportInput struct {
	Provider            string
	FileFingerprint     string
	StatementFrom       time.Time
	StatementTo         time.Time
	WalletID            string
	CreateWallet        *CreateWalletForImport
	OpeningBalanceCents int64
	EndingBalanceCents  int64
	Reconciliation      string
	Rows                []ImportRowInput
}

// Create commits an import batch atomically. Re-importing the same file
// fingerprint replays the existing batch instead of duplicating rows.
func (s *ImportService) Create(ctx context.Context, userEmail string, input CreateImportInput) (*domain.ImportBatch, error) {
	if len(input.Rows) > domain.MaxImportRows {
		return nil, fmt.Errorf("too many rows (max %d)", domain.MaxImportRows)
	}
	if input.WalletID == "" && input.CreateWallet == nil {
		return nil, fmt.Errorf("a wallet or create-wallet request is required")
	}
	switch input.Reconciliation {
	case domain.ReconciliationOK, domain.ReconciliationMismatch, domain.ReconciliationUnknown:
	default:
		return nil, fmt.Errorf("invalid reconciliation state: %s", input.Reconciliation)
	}

	batch, err := domain.NewImportBatch(
		uuid.New().String(), userEmail, input.Provider, input.FileFingerprint,
		input.StatementFrom, input.StatementTo, input.OpeningBalanceCents, input.EndingBalanceCents,
	)
	if err != nil {
		return nil, err
	}
	batch.Reconciliation = input.Reconciliation

	// Fast path: an identical file was already imported.
	existing, err := s.importRepo.FindByFingerprint(ctx, userEmail, input.FileFingerprint)
	if err != nil {
		return nil, fmt.Errorf("check existing import: %w", err)
	}
	if existing != nil {
		existing.Summary.Replay = true
		return existing, nil
	}

	currency, err := ImportProviderCurrency(input.Provider)
	if err != nil {
		return nil, err
	}

	err = s.coordinator.WithTx(ctx, func(txCtx context.Context) error {
		// Resolve or create the target wallet inside the transaction.
		var wallet *domain.Wallet
		if input.WalletID != "" {
			wallet, err = s.walletRepo.FindByID(txCtx, input.WalletID)
			if err != nil {
				return fmt.Errorf("find wallet: %w", err)
			}
			if wallet.UserEmail != userEmail {
				return fmt.Errorf("wallet not found")
			}
			if wallet.ArchivedAt != nil {
				return fmt.Errorf("wallet is archived")
			}
			if wallet.Currency != currency {
				return fmt.Errorf("wallet currency %s does not match import currency %s", wallet.Currency, currency)
			}
		} else {
			existingWallets, err := s.walletRepo.ListByUser(txCtx, userEmail)
			if err != nil {
				return fmt.Errorf("list wallets: %w", err)
			}
			wallet, err = domain.NewWallet(
				uuid.New().String(), userEmail, input.CreateWallet.Name,
				domain.WalletEwallet, currency, input.CreateWallet.OpeningBalanceCents,
				len(existingWallets) == 0,
			)
			if err != nil {
				return err
			}
			if err := s.walletRepo.Save(txCtx, wallet); err != nil {
				return fmt.Errorf("save wallet: %w", err)
			}
			batch.CreatedWalletID = wallet.ID
		}
		batch.WalletID = wallet.ID

		entries := make([]*domain.ImportEntry, 0, len(input.Rows))
		seen := make(map[string]bool, len(input.Rows))

		for _, row := range input.Rows {
			entry, err := domain.NewImportEntry(
				uuid.New().String(), batch.ID, row.SourceReference, row.OccurredAt,
				row.AmountCents, row.Kind, row.Category, row.Description,
				row.Counterparty, row.CounterWalletID,
			)
			if err != nil {
				return fmt.Errorf("row %s: %w", row.SourceReference, err)
			}

			if seen[row.SourceReference] {
				entry.Outcome = domain.EntryOutcomeDuplicate
				batch.Summary.Duplicates++
				entries = append(entries, entry)
				continue
			}
			seen[row.SourceReference] = true

			idemKey := "imp:" + batch.ID + ":" + row.SourceReference
			txDate := row.OccurredAt

			switch row.Kind {
			case domain.EntryExpense, domain.EntryIncome:
				txType := domain.TransactionExpense
				if row.Kind == domain.EntryIncome {
					txType = domain.TransactionIncome
				}
				tx, err := domain.NewTransaction(
					uuid.New().String(), userEmail, wallet.Currency, row.Category,
					row.Description, row.AmountCents, txType, txDate,
				)
				if err != nil {
					return err
				}
				tx.WalletID = wallet.ID
				tx.IdempotencyKey = idemKey
				if err := s.txRepo.Save(txCtx, tx); err != nil {
					return fmt.Errorf("save transaction: %w", err)
				}
				entry.EntityType = "transaction"
				entry.EntityID = tx.ID
				batch.Summary.Transactions++
				if txType == domain.TransactionIncome {
					batch.Summary.IncomeCents += tx.AmountCents
				} else {
					batch.Summary.ExpenseCents += tx.AmountCents
				}
			case domain.EntryTransferOut, domain.EntryTransferIn:
				fromID, toID := wallet.ID, row.CounterWalletID
				if row.Kind == domain.EntryTransferIn {
					fromID, toID = row.CounterWalletID, wallet.ID
				}
				if fromID == toID {
					return fmt.Errorf("row %s: transfer requires a different counter wallet", row.SourceReference)
				}
				transfer, err := domain.NewWalletTransfer(
					uuid.New().String(), userEmail, fromID, toID, row.Description,
					row.AmountCents, row.AmountCents, txDate,
				)
				if err != nil {
					return err
				}
				transfer.IdempotencyKey = idemKey
				if err := s.transferRepo.Save(txCtx, transfer); err != nil {
					return fmt.Errorf("save transfer: %w", err)
				}
				entry.EntityType = "transfer"
				entry.EntityID = transfer.ID
				batch.Summary.Transfers++
			case domain.EntrySkipped:
				entry.Outcome = domain.EntryOutcomeExcluded
				batch.Summary.Excluded++
			default:
				entry.Outcome = domain.EntryOutcomeError
				batch.Summary.Errors++
			}

			entries = append(entries, entry)
		}

		batch.Summary.Total = len(input.Rows)
		batch.Summary.Imported = batch.Summary.Transactions + batch.Summary.Transfers

		if err := s.importRepo.SaveBatch(txCtx, batch); err != nil {
			return err
		}
		return s.importRepo.SaveEntries(txCtx, entries)
	})
	if err != nil {
		return nil, err
	}

	return batch, nil
}

// Rollback undoes an import: it deletes the entities this batch created and
// removes the wallet when the batch created it and nothing else references it.
// Rollback is idempotent — a second call on an already rolled-back batch is a
// no-op.
func (s *ImportService) Rollback(ctx context.Context, id, userEmail string) (int, error) {
	batch, err := s.importRepo.FindBatchByID(ctx, id, userEmail)
	if err != nil {
		return 0, err
	}
	if batch.Status == domain.ImportStatusRolledBack {
		return 0, nil
	}

	removed := 0
	err = s.coordinator.WithTx(ctx, func(txCtx context.Context) error {
		entries, err := s.importRepo.ListEntries(txCtx, id)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Outcome != domain.EntryOutcomeImported || e.EntityType == "" {
				continue
			}
			if err := s.importRepo.DeleteTransactionEntity(txCtx, e.EntityType, e.EntityID, userEmail); err != nil {
				return err
			}
			removed++
		}

		if batch.CreatedWalletID != "" {
			txCount, err := s.importRepo.CountTransactionsForWallet(txCtx, batch.CreatedWalletID)
			if err != nil {
				return err
			}
			transferCount, err := s.importRepo.CountTransfersForWallet(txCtx, batch.CreatedWalletID)
			if err != nil {
				return err
			}
			if txCount == 0 && transferCount == 0 {
				if err := s.importRepo.DeleteWallet(txCtx, batch.CreatedWalletID, userEmail); err != nil {
					return err
				}
			}
		}

		return s.importRepo.MarkRolledBack(txCtx, id, time.Now().UTC())
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

type ImportFilter struct {
	Limit  int
	Offset int
}

func (s *ImportService) List(ctx context.Context, userEmail string, filter ImportFilter) ([]*domain.ImportBatch, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.importRepo.ListByUser(ctx, userEmail, filter.Limit, filter.Offset)
}

func (s *ImportService) Get(ctx context.Context, id, userEmail string) (*domain.ImportBatch, []*domain.ImportEntry, error) {
	batch, err := s.importRepo.FindBatchByID(ctx, id, userEmail)
	if err != nil {
		return nil, nil, err
	}
	entries, err := s.importRepo.ListEntries(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return batch, entries, nil
}
