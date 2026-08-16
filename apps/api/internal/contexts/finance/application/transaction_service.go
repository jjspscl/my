package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// Used to detect idempotent replay races after a concurrent insert won the
// unique-index race.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// BillAutoMatcher is an optional hook to auto-match bills after a transaction is created.
type BillAutoMatcher interface {
	TryAutoMatch(ctx context.Context, tx *domain.Transaction)
}

// TransactionBillLinkRepo exposes the bill-payment side effects of editing or
// deleting a transaction.
type TransactionBillLinkRepo interface {
	FindPaymentsByTransaction(ctx context.Context, txID string) ([]*domain.BillPayment, error)
	DeletePayment(ctx context.Context, paymentID string) error
}

// ImportProvenanceMarker records edits/deletes of imported transactions on
// their immutable statement entries.
type ImportProvenanceMarker interface {
	MarkTransactionProvenance(ctx context.Context, txID, status string, at time.Time) error
}

type TransactionService struct {
	repo         domain.TransactionRepository
	walletRepo   domain.WalletRepository
	clock        *timeutil.Clock
	billMatcher  BillAutoMatcher
	coordinator  TxCoordinator
	billLinks    TransactionBillLinkRepo
	importMarker ImportProvenanceMarker
}

func NewTransactionService(repo domain.TransactionRepository, walletRepo domain.WalletRepository, clock *timeutil.Clock) *TransactionService {
	return &TransactionService{repo: repo, walletRepo: walletRepo, clock: clock}
}

// WithBillAutoMatcher sets the bill auto-matcher hook.
func (s *TransactionService) WithBillAutoMatcher(m BillAutoMatcher) *TransactionService {
	s.billMatcher = m
	return s
}

// WithCoordinator makes transaction mutations and their side effects atomic.
func (s *TransactionService) WithCoordinator(c TxCoordinator) *TransactionService {
	s.coordinator = c
	return s
}

// WithBillLinkRepo wires bill-payment reconciliation for edits/deletes.
func (s *TransactionService) WithBillLinkRepo(r TransactionBillLinkRepo) *TransactionService {
	s.billLinks = r
	return s
}

// WithImportProvenanceMarker wires import lifecycle tracking for edits/deletes.
func (s *TransactionService) WithImportProvenanceMarker(m ImportProvenanceMarker) *TransactionService {
	s.importMarker = m
	return s
}

type CreateTransactionInput struct {
	AmountCents     int64
	Category        string
	Description     string
	Type            domain.TransactionType
	WalletID        string
	TransactionDate time.Time
	IdempotencyKey  string
}

func (s *TransactionService) resolveWallet(ctx context.Context, userEmail, walletID string) (*domain.Wallet, error) {
	if walletID == "" {
		defaultWallet, err := s.walletRepo.FindDefault(ctx, userEmail)
		if err != nil {
			return nil, fmt.Errorf("no wallet specified and no default wallet found: %w", err)
		}
		return defaultWallet, nil
	}

	return ensureUsableWallet(ctx, s.walletRepo, userEmail, walletID)
}

func (s *TransactionService) Create(ctx context.Context, userEmail string, input CreateTransactionInput) (*domain.Transaction, error) {
	if input.IdempotencyKey != "" {
		if len(input.IdempotencyKey) > domain.MaxIdempotencyLen {
			return nil, fmt.Errorf("idempotency key too long (max %d)", domain.MaxIdempotencyLen)
		}
		existing, err := s.repo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("check idempotency: %w", err)
		}
		if existing != nil {
			return existing, nil
		}
	}

	wallet, err := s.resolveWallet(ctx, userEmail, input.WalletID)
	if err != nil {
		return nil, err
	}

	// Wallet is the currency authority: a transaction is denominated in the
	// currency of the wallet it is booked against, never in a global default.
	tx, err := domain.NewTransaction(
		uuid.New().String(),
		userEmail,
		wallet.Currency,
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
	tx.IdempotencyKey = input.IdempotencyKey

	if err := s.repo.Save(ctx, tx); err != nil {
		if input.IdempotencyKey != "" && isUniqueViolation(err) {
			// A concurrent replay won the unique-index race; return its row.
			if existing, ferr := s.repo.FindByIdempotencyKey(ctx, userEmail, input.IdempotencyKey); ferr == nil && existing != nil {
				return existing, nil
			}
		}
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

// TodayTotals returns today's income/expense/net totals grouped by currency,
// resolved against the user's financial calendar, never the server's UTC clock.
// When the day has no transactions the slice is empty; callers render zeros.
func (s *TransactionService) TodayTotals(ctx context.Context, userEmail string) ([]domain.CurrencyTotal, error) {
	totals, err := s.repo.GetTodayTotals(ctx, userEmail, s.clock.TodayStart())
	if err != nil {
		return nil, fmt.Errorf("get today totals: %w", err)
	}
	return totals, nil
}

// GetTodayTotal returns the single-currency daily total. For backwards
// compatibility it prefers the default currency and falls back to the first
// currency present. New code should use TodayTotals for per-currency results.
func (s *TransactionService) GetTodayTotal(ctx context.Context, userEmail, defaultCurrency string) (*domain.DailyTotal, error) {
	totals, err := s.TodayTotals(ctx, userEmail)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	total := &domain.DailyTotal{
		Date:     now.Format("2006-01-02"),
		Currency: defaultCurrency,
	}

	for _, t := range totals {
		if t.Currency == defaultCurrency {
			total.TotalCents = t.TotalCents
			total.ExpenseCents = t.ExpenseCents
			total.IncomeCents = t.IncomeCents
			return total, nil
		}
	}

	if len(totals) > 0 {
		total.Currency = totals[0].Currency
		total.TotalCents = totals[0].TotalCents
		total.ExpenseCents = totals[0].ExpenseCents
		total.IncomeCents = totals[0].IncomeCents
	}

	return total, nil
}

// UpdateTransactionInput is a partial update: nil fields stay unchanged. At
// least one field must be set (enforced by the caller).
type UpdateTransactionInput struct {
	AmountCents     *int64
	Category        *string
	Description     *string
	Type            *domain.TransactionType
	WalletID        *string
	TransactionDate *time.Time
	// ExpectedRevision is the revision the client last saw; when > 0 the
	// update is rejected with domain.ErrStaleRevision if the stored row has
	// moved past it.
	ExpectedRevision int
}

// Update applies a partial edit to a transaction, atomically reconciling
// auto-matched bill payments and import provenance. Wallet is the currency
// authority: moving a transaction to another wallet re-denominates it.
func (s *TransactionService) Update(ctx context.Context, userEmail, id string, input UpdateTransactionInput) (*domain.Transaction, error) {
	if input.AmountCents == nil && input.Category == nil && input.Description == nil &&
		input.Type == nil && input.WalletID == nil && input.TransactionDate == nil {
		return nil, fmt.Errorf("empty update: no fields provided")
	}

	var updated *domain.Transaction
	run := func(txCtx context.Context) error {
		tx, err := s.updateOne(txCtx, userEmail, id, input)
		if err != nil {
			return err
		}
		updated = tx
		return nil
	}

	if s.coordinator != nil {
		if err := s.coordinator.WithTx(ctx, run); err != nil {
			return nil, err
		}
	} else if err := run(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// updateOne performs the edit against ctx (which may or may not already carry
// a transaction from the coordinator). Shared by Update and UpdateMany; the
// caller owns transaction boundaries.
func (s *TransactionService) updateOne(ctx context.Context, userEmail, id string, input UpdateTransactionInput) (*domain.Transaction, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.UserEmail != userEmail {
		return nil, fmt.Errorf("transaction not found")
	}
	if input.ExpectedRevision > 0 && existing.Revision != input.ExpectedRevision {
		return nil, domain.ErrStaleRevision
	}

	amountCents := existing.AmountCents
	category := existing.Category
	description := existing.Description
	txType := existing.Type
	txDate := existing.TransactionDate
	walletID := existing.WalletID
	currency := existing.Currency

	if input.AmountCents != nil {
		amountCents = *input.AmountCents
	}
	if input.Category != nil {
		category = *input.Category
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.Type != nil {
		txType = *input.Type
	}
	if input.TransactionDate != nil {
		txDate = *input.TransactionDate
	}
	if input.WalletID != nil && *input.WalletID != walletID {
		wallet, err := ensureUsableWallet(ctx, s.walletRepo, userEmail, *input.WalletID)
		if err != nil {
			return nil, err
		}
		walletID = wallet.ID
		currency = wallet.Currency
	}

	tx, err := domain.NewTransaction(id, userEmail, currency, category, description, amountCents, txType, txDate)
	if err != nil {
		return nil, err
	}
	tx.WalletID = walletID
	tx.CreatedAt = existing.CreatedAt
	tx.IdempotencyKey = existing.IdempotencyKey
	tx.Revision = existing.Revision
	tx.Imported = existing.Imported
	tx.ImportProvider = existing.ImportProvider
	tx.UpdatedAt = s.clock.Now()

	if err := s.reconcileAutoBillLinks(ctx, tx, existing); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, tx, tx.Revision); err != nil {
		return nil, err
	}
	if tx.Imported && s.importMarker != nil {
		if err := s.importMarker.MarkTransactionProvenance(ctx, tx.ID, domain.EntityStatusModified, s.clock.Now()); err != nil {
			return nil, fmt.Errorf("mark import entry modified: %w", err)
		}
	}
	// Re-evaluate auto-match against the new values; it only creates a
	// payment when the updated transaction still matches a bill.
	if s.billMatcher != nil && txType == domain.TransactionExpense {
		s.billMatcher.TryAutoMatch(ctx, tx)
	}

	tx.Revision++
	return tx, nil
}

// MaxBulkTransactionItems caps how many transactions one bulk request may
// touch, bounding the time a single transaction stays open.
const MaxBulkTransactionItems = 200

// BulkItem identifies one transaction in a bulk request together with the
// revision the client last saw (per-row optimistic concurrency).
type BulkItem struct {
	ID       string
	Revision int
}

// BulkPatch is the shared partial edit applied to every item. Only fields
// that are safe to set uniformly across many rows are allowed: amount, date,
// and description are deliberately excluded because mass-applying them to
// unrelated rows is almost always a mistake.
type BulkPatch struct {
	Category *string
	Type     *domain.TransactionType
	WalletID *string
}

func (p BulkPatch) empty() bool {
	return p.Category == nil && p.Type == nil && p.WalletID == nil
}

// UpdateMany applies one patch to many transactions atomically: all items
// succeed or none do. Per-row revision preconditions and every per-item side
// effect (bill-link reconciliation, import provenance) behave exactly like
// single edits.
func (s *TransactionService) UpdateMany(ctx context.Context, userEmail string, items []BulkItem, patch BulkPatch) ([]*domain.Transaction, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("empty bulk update: no items")
	}
	if len(items) > MaxBulkTransactionItems {
		return nil, fmt.Errorf("bulk update exceeds limit of %d transactions", MaxBulkTransactionItems)
	}
	if patch.empty() {
		return nil, fmt.Errorf("empty bulk update: no fields provided")
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it.ID == "" {
			return nil, fmt.Errorf("bulk update item missing id")
		}
		if _, dup := seen[it.ID]; dup {
			return nil, fmt.Errorf("bulk update contains duplicate transaction %s", it.ID)
		}
		seen[it.ID] = struct{}{}
	}

	var updated []*domain.Transaction
	run := func(txCtx context.Context) error {
		for _, it := range items {
			tx, err := s.updateOne(txCtx, userEmail, it.ID, UpdateTransactionInput{
				Category:         patch.Category,
				Type:             patch.Type,
				WalletID:         patch.WalletID,
				ExpectedRevision: it.Revision,
			})
			if err != nil {
				return err
			}
			updated = append(updated, tx)
		}
		return nil
	}

	if s.coordinator != nil {
		if err := s.coordinator.WithTx(ctx, run); err != nil {
			return nil, err
		}
	} else if err := run(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteMany removes many transactions atomically. Revision preconditions and
// side-effect reconciliation behave like single deletes; returns the count of
// removed transactions.
func (s *TransactionService) DeleteMany(ctx context.Context, userEmail string, items []BulkItem) (int, error) {
	if len(items) == 0 {
		return 0, fmt.Errorf("empty bulk delete: no items")
	}
	if len(items) > MaxBulkTransactionItems {
		return 0, fmt.Errorf("bulk delete exceeds limit of %d transactions", MaxBulkTransactionItems)
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it.ID == "" {
			return 0, fmt.Errorf("bulk delete item missing id")
		}
		if _, dup := seen[it.ID]; dup {
			return 0, fmt.Errorf("bulk delete contains duplicate transaction %s", it.ID)
		}
		seen[it.ID] = struct{}{}
	}

	deleted := 0
	run := func(txCtx context.Context) error {
		for _, it := range items {
			if err := s.deleteOne(txCtx, it.ID, userEmail, it.Revision, it.Revision > 0); err != nil {
				return err
			}
			deleted++
		}
		return nil
	}

	if s.coordinator != nil {
		if err := s.coordinator.WithTx(ctx, run); err != nil {
			return 0, err
		}
	} else if err := run(ctx); err != nil {
		return 0, err
	}
	return deleted, nil
}

// matchAffecting reports whether a change could invalidate an auto-matched
// bill payment (amount, category, type, or date).
func matchAffecting(prev, next *domain.Transaction) bool {
	return prev.AmountCents != next.AmountCents ||
		prev.Category != next.Category ||
		prev.Type != next.Type ||
		prev.TransactionDate.Format("2006-01-02") != next.TransactionDate.Format("2006-01-02")
}

// reconcileAutoBillLinks removes bill payments that were inferred by auto-match
// when the underlying transaction changed; explicit (manual/legacy) links are
// preserved so a user-marked-paid bill stays paid.
func (s *TransactionService) reconcileAutoBillLinks(ctx context.Context, next, prev *domain.Transaction) error {
	if s.billLinks == nil || !matchAffecting(prev, next) {
		return nil
	}
	return s.removeAutoBillLinks(ctx, next.ID)
}

// removeAutoBillLinks deletes every auto-matched bill payment referencing the
// transaction. Used when the transaction changed or is being deleted.
func (s *TransactionService) removeAutoBillLinks(ctx context.Context, txID string) error {
	if s.billLinks == nil {
		return nil
	}
	payments, err := s.billLinks.FindPaymentsByTransaction(ctx, txID)
	if err != nil {
		return fmt.Errorf("find linked bill payments: %w", err)
	}
	for _, p := range payments {
		if p.TransactionLinkSource == domain.PaymentLinkAuto {
			if err := s.billLinks.DeletePayment(ctx, p.ID); err != nil {
				return fmt.Errorf("remove auto-matched bill payment: %w", err)
			}
		}
	}
	return nil
}

func (s *TransactionService) Delete(ctx context.Context, id, userEmail string) error {
	return s.deleteOneWrapped(ctx, id, userEmail, 0, false)
}

// DeleteAtRevision deletes only when the client's revision still matches the
// stored one, returning domain.ErrStaleRevision otherwise.
func (s *TransactionService) DeleteAtRevision(ctx context.Context, id, userEmail string, revision int) error {
	return s.deleteOneWrapped(ctx, id, userEmail, revision, true)
}

// deleteOneWrapped is the single-item public delete path: one transaction
// boundary around deleteOne.
func (s *TransactionService) deleteOneWrapped(ctx context.Context, id, userEmail string, revision int, enforceRevision bool) error {
	run := func(txCtx context.Context) error {
		return s.deleteOne(txCtx, id, userEmail, revision, enforceRevision)
	}
	if s.coordinator != nil {
		return s.coordinator.WithTx(ctx, run)
	}
	return run(ctx)
}

// deleteOne removes one transaction and its side effects against ctx (which
// may already carry a coordinator transaction): auto-matched bill payments
// are deleted (explicit links stay paid via FK SET NULL), and imported
// entries are marked deleted so rollback and reconciliation stay accurate.
func (s *TransactionService) deleteOne(ctx context.Context, id, userEmail string, revision int, enforceRevision bool) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.UserEmail != userEmail {
		return fmt.Errorf("transaction not found")
	}
	if enforceRevision && existing.Revision != revision {
		return domain.ErrStaleRevision
	}

	if err := s.removeAutoBillLinks(ctx, id); err != nil {
		return err
	}
	if existing.Imported && s.importMarker != nil {
		if err := s.importMarker.MarkTransactionProvenance(ctx, id, domain.EntityStatusDeleted, s.clock.Now()); err != nil {
			return fmt.Errorf("mark import entry deleted: %w", err)
		}
	}
	if enforceRevision {
		return s.repo.DeleteAtRevision(ctx, id, userEmail, revision)
	}
	return s.repo.Delete(ctx, id, userEmail)
}
