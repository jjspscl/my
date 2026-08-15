package domain

import (
	"context"
	"fmt"
	"time"
)

// Import providers.
const (
	ImportProviderGCashPDF = "gcash_pdf"
)

// Reconciliation states for a statement import.
const (
	ReconciliationOK      = "ok"
	ReconciliationMismatch = "mismatch"
	ReconciliationUnknown  = "unknown"
)

// Import statuses.
const (
	ImportStatusCompleted   = "completed"
	ImportStatusRolledBack  = "rolled_back"
)

// Entry kinds — how a statement row is booked.
const (
	EntryExpense       = "expense"
	EntryIncome        = "income"
	EntryTransferOut   = "transfer_out" // money left the imported wallet
	EntryTransferIn    = "transfer_in"  // money arrived in the imported wallet
	EntrySkipped       = "skipped"
)

// Entry outcomes.
const (
	EntryOutcomeImported  = "imported"
	EntryOutcomeDuplicate = "duplicate"
	EntryOutcomeExcluded  = "excluded"
	EntryOutcomeError     = "error"
)

// Limits for validated import payloads.
const (
	MaxImportRows         = 2000
	MaxImportRefLen       = 100
	MaxImportDescLen      = 500
	MaxImportCategoryLen  = MaxCategoryLen
	MaxImportCounterparty = 200
)

type ImportBatch struct {
	ID                  string
	UserEmail           string
	Provider            string
	FileFingerprint     string
	StatementFrom       time.Time
	StatementTo         time.Time
	WalletID            string
	CreatedWalletID     string
	OpeningBalanceCents int64
	EndingBalanceCents  int64
	Reconciliation      string
	Status              string
	Summary             ImportSummary
	CreatedAt           time.Time
	RolledBackAt        *time.Time
}

type ImportSummary struct {
	Total        int   `json:"total"`
	Imported     int   `json:"imported"`
	Duplicates   int   `json:"duplicates"`
	Excluded     int   `json:"excluded"`
	Errors       int   `json:"errors"`
	Transactions int   `json:"transactions"`
	Transfers    int   `json:"transfers"`
	IncomeCents  int64 `json:"incomeCents"`
	ExpenseCents int64 `json:"expenseCents"`
	Replay       bool  `json:"replay"`
}

// ImportEntry is one statement row as booked. Kind and outcome are set by the
// client's review step; the server validates them before committing.
type ImportEntry struct {
	ID              string
	ImportID        string
	SourceReference string
	OccurredAt      time.Time
	AmountCents     int64
	Kind            string
	Category        string
	Description     string
	Counterparty    string
	CounterWalletID string
	Outcome         string
	EntityType      string
	EntityID        string
}

// NewImportBatch validates batch-level invariants.
func NewImportBatch(id, userEmail, provider, fileFingerprint string, statementFrom, statementTo time.Time, openingBalanceCents, endingBalanceCents int64) (*ImportBatch, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if provider != ImportProviderGCashPDF {
		return nil, fmt.Errorf("unsupported import provider: %s", provider)
	}
	if len(fileFingerprint) != 64 {
		return nil, fmt.Errorf("file fingerprint must be a sha256 hex digest")
	}
	if statementFrom.IsZero() || statementTo.IsZero() {
		return nil, fmt.Errorf("statement range is required")
	}
	if statementTo.Before(statementFrom) {
		return nil, fmt.Errorf("statement range is inverted")
	}
	if openingBalanceCents < 0 || endingBalanceCents < 0 {
		return nil, fmt.Errorf("balances cannot be negative")
	}

	return &ImportBatch{
		ID:                  id,
		UserEmail:           userEmail,
		Provider:            provider,
		FileFingerprint:     fileFingerprint,
		StatementFrom:       statementFrom,
		StatementTo:         statementTo,
		OpeningBalanceCents: openingBalanceCents,
		EndingBalanceCents:  endingBalanceCents,
		Reconciliation:      ReconciliationUnknown,
		Status:              ImportStatusCompleted,
		CreatedAt:           time.Now().UTC(),
	}, nil
}

// NewImportEntry validates a single statement row before booking.
func NewImportEntry(id, importID, sourceReference string, occurredAt time.Time, amountCents int64, kind, category, description, counterparty, counterWalletID string) (*ImportEntry, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if importID == "" {
		return nil, fmt.Errorf("import id is required")
	}
	if sourceReference == "" || len(sourceReference) > MaxImportRefLen {
		return nil, fmt.Errorf("source reference is required (max %d)", MaxImportRefLen)
	}
	if occurredAt.IsZero() {
		return nil, fmt.Errorf("occurred at is required")
	}
	if amountCents <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}
	switch kind {
	case EntryExpense, EntryIncome, EntryTransferOut, EntryTransferIn, EntrySkipped:
	default:
		return nil, fmt.Errorf("invalid entry kind: %s", kind)
	}
	if kind != EntrySkipped {
		if category == "" || len(category) > MaxImportCategoryLen {
			return nil, fmt.Errorf("category is required (max %d)", MaxImportCategoryLen)
		}
		if len(description) > MaxImportDescLen {
			return nil, fmt.Errorf("description too long (max %d)", MaxImportDescLen)
		}
	}
	if len(counterparty) > MaxImportCounterparty {
		return nil, fmt.Errorf("counterparty too long (max %d)", MaxImportCounterparty)
	}
	if (kind == EntryTransferIn || kind == EntryTransferOut) && counterWalletID == "" {
		return nil, fmt.Errorf("transfer entries require a counter wallet")
	}
	if (kind == EntryExpense || kind == EntryIncome) && counterWalletID != "" {
		return nil, fmt.Errorf("non-transfer entries cannot carry a counter wallet")
	}

	return &ImportEntry{
		ID:              id,
		ImportID:        importID,
		SourceReference: sourceReference,
		OccurredAt:      occurredAt,
		AmountCents:     amountCents,
		Kind:            kind,
		Category:        category,
		Description:     description,
		Counterparty:    counterparty,
		CounterWalletID: counterWalletID,
		Outcome:         EntryOutcomeImported,
	}, nil
}

// ImportRepository persists import batches and their entries. Every method
// honors the transaction coordinator via executor-style context plumbing.
type ImportRepository interface {
	SaveBatch(ctx context.Context, batch *ImportBatch) error
	SaveEntries(ctx context.Context, entries []*ImportEntry) error
	FindByFingerprint(ctx context.Context, userEmail, fingerprint string) (*ImportBatch, error)
	FindBatchByID(ctx context.Context, id, userEmail string) (*ImportBatch, error)
	ListByUser(ctx context.Context, userEmail string, limit, offset int) ([]*ImportBatch, error)
	ListEntries(ctx context.Context, importID string) ([]*ImportEntry, error)
	MarkRolledBack(ctx context.Context, id string, rolledBackAt time.Time) error
	DeleteTransactionEntity(ctx context.Context, entityType, entityID, userEmail string) error
	DeleteWallet(ctx context.Context, id, userEmail string) error
	CountTransactionsForWallet(ctx context.Context, walletID string) (int, error)
	CountTransfersForWallet(ctx context.Context, walletID string) (int, error)
}
