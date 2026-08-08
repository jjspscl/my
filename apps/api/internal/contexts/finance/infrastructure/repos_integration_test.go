package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

const testUser = "user@example.com"

func mustWallet(t *testing.T, repo *WalletRepoLibSQL, id, currency string, opening int64) *domain.Wallet {
	t.Helper()
	w, err := domain.NewWallet(id, testUser, id, domain.WalletBank, currency, opening, false)
	if err != nil {
		t.Fatalf("new wallet %s: %v", id, err)
	}
	if err := repo.Save(context.Background(), w); err != nil {
		t.Fatalf("save wallet %s: %v", id, err)
	}
	return w
}

func mustTransaction(t *testing.T, repo *TransactionRepoLibSQL, id, currency, category string, amount int64, txType domain.TransactionType, date time.Time, walletID string) {
	t.Helper()
	tx, err := domain.NewTransaction(id, testUser, currency, category, "test", amount, txType, date)
	if err != nil {
		t.Fatalf("new transaction %s: %v", id, err)
	}
	tx.WalletID = walletID
	if err := repo.Save(context.Background(), tx); err != nil {
		t.Fatalf("save transaction %s: %v", id, err)
	}
}

// TestTransferDualLegsAndLegacyFallback pins the migration contract: rows
// written before 009 have NULL from/to legs and must read back as the legacy
// single amount; rows written after 009 keep their distinct legs.
func TestTransferDualLegsAndLegacyFallback(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustWallet(t, walletRepo, "w-usd", "USD", 0)

	// Legacy row: only amount_cents populated, legs NULL.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO wallet_transfers (id, user_email, from_wallet_id, to_wallet_id, amount_cents, description, transfer_date, created_at)
		VALUES ('legacy-1', ?, 'w-php', 'w-usd', 5000, 'old transfer', '2026-01-01', datetime('now'))
	`, testUser); err != nil {
		t.Fatalf("insert legacy transfer: %v", err)
	}

	legacy, err := transferRepo.FindByID(ctx, "legacy-1")
	if err != nil {
		t.Fatalf("find legacy transfer: %v", err)
	}
	if legacy.FromAmountCents != 5000 || legacy.ToAmountCents != 5000 {
		t.Errorf("legacy fallback: got from=%d to=%d, want 5000/5000", legacy.FromAmountCents, legacy.ToAmountCents)
	}

	// Modern row: distinct legs survive the round trip.
	modern, err := domain.NewWalletTransfer("modern-1", testUser, "w-php", "w-usd", "cross-currency", 10000, 5000, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new transfer: %v", err)
	}
	if err := transferRepo.Save(ctx, modern); err != nil {
		t.Fatalf("save modern transfer: %v", err)
	}
	got, err := transferRepo.FindByID(ctx, "modern-1")
	if err != nil {
		t.Fatalf("find modern transfer: %v", err)
	}
	if got.FromAmountCents != 10000 || got.ToAmountCents != 5000 {
		t.Errorf("modern legs: got from=%d to=%d, want 10000/5000", got.FromAmountCents, got.ToAmountCents)
	}
}

// TestIdempotencyUniqueIndexes pins the migration contract: the partial unique
// indexes reject a second row with the same key, and isUniqueViolation
// recognizes the failure so services can treat it as a replay race.
func TestIdempotencyUniqueIndexes(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)
	goalRepo := NewGoalRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustWallet(t, walletRepo, "w-php2", "PHP", 0)

	// Transactions.
	mustTransaction(t, txRepo, "tx-1", "PHP", "Food", 1000, domain.TransactionExpense, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "w-php")
	dup, err := domain.NewTransaction("tx-2", testUser, "PHP", "Food", "dup", 1000, domain.TransactionExpense, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new dup transaction: %v", err)
	}
	dup.WalletID = "w-php"
	dup.IdempotencyKey = "key-tx"
	if err := txRepo.Save(ctx, dup); err != nil {
		t.Fatalf("save first transaction with key: %v", err)
	}
	dup2 := *dup
	dup2.ID = "tx-3"
	if err := txRepo.Save(ctx, &dup2); err == nil {
		t.Fatal("expected duplicate idempotency key to fail")
	} else if !isUniqueViolation(err) {
		t.Fatalf("expected unique violation, got: %v", err)
	}

	// Transfers.
	tr, err := domain.NewWalletTransfer("tr-1", "tr-2", "w-php", "w-php2", "x", 100, 100, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new transfer: %v", err)
	}
	tr.IdempotencyKey = "key-tr"
	if err := transferRepo.Save(ctx, tr); err != nil {
		t.Fatalf("save first transfer with key: %v", err)
	}
	tr2 := *tr
	tr2.ID = "tr-2"
	if err := transferRepo.Save(ctx, &tr2); err == nil || !isUniqueViolation(err) {
		t.Fatalf("expected unique violation for transfer, got: %v", err)
	}

	// Goal contributions.
	goal, err := domain.NewSavingsGoal("goal-1", testUser, "Trip", 100000, nil, "w-php", "PHP")
	if err != nil {
		t.Fatalf("new goal: %v", err)
	}
	if err := goalRepo.SaveGoal(ctx, goal); err != nil {
		t.Fatalf("save goal: %v", err)
	}
	note := "first"
	contrib, err := domain.NewGoalContribution("c-1", "goal-1", 1000, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &note, nil, nil)
	if err != nil {
		t.Fatalf("new contribution: %v", err)
	}
	contrib.IdempotencyKey = "key-c"
	if err := goalRepo.SaveContribution(ctx, contrib); err != nil {
		t.Fatalf("save first contribution with key: %v", err)
	}
	contrib2 := *contrib
	contrib2.ID = "c-2"
	if err := goalRepo.SaveContribution(ctx, &contrib2); err == nil || !isUniqueViolation(err) {
		t.Fatalf("expected unique violation for contribution, got: %v", err)
	}
}

// TestGetBalancesByUserPerLeg verifies the balance math uses the per-leg
// transfer amounts: a PHP→USD transfer debits the PHP wallet by the from leg
// and credits the USD wallet by the to leg, never by a single shared amount.
func TestGetBalancesByUserPerLeg(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 100000)
	mustWallet(t, walletRepo, "w-usd", "USD", 0)

	tr, err := domain.NewWalletTransfer("tr-1", "tr-1", "w-php", "w-usd", "convert", 10000, 5000, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new transfer: %v", err)
	}
	if err := transferRepo.Save(ctx, tr); err != nil {
		t.Fatalf("save transfer: %v", err)
	}

	balances, err := walletRepo.GetBalancesByUser(ctx, testUser)
	if err != nil {
		t.Fatalf("get balances: %v", err)
	}
	byID := map[string]*domain.WalletBalance{}
	for _, b := range balances {
		byID[b.Wallet.ID] = b
	}

	php, ok := byID["w-php"]
	if !ok {
		t.Fatal("missing PHP wallet balance")
	}
	if php.BalanceCents != 90000 {
		t.Errorf("PHP balance = %d, want 90000 (100000 - 10000 from leg)", php.BalanceCents)
	}
	usd, ok := byID["w-usd"]
	if !ok {
		t.Fatal("missing USD wallet balance")
	}
	if usd.BalanceCents != 5000 {
		t.Errorf("USD balance = %d, want 5000 (to leg)", usd.BalanceCents)
	}
}

// TestGetSpentByCategoryHalfOpenRange pins the month-boundary contract: the
// predicate is transaction_date >= from AND transaction_date < to, so the last
// day of the month counts and the first day of the next month does not.
func TestGetSpentByCategoryHalfOpenRange(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	budgetRepo := NewBudgetRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustTransaction(t, txRepo, "tx-feb", "PHP", "Food", 1000, domain.TransactionExpense, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), "w-php")
	mustTransaction(t, txRepo, "tx-mar", "PHP", "Food", 2000, domain.TransactionExpense, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "w-php")

	clock := timeutil.New(time.UTC)
	from, to, err := clock.MonthRange("2026-02")
	if err != nil {
		t.Fatalf("month range: %v", err)
	}

	spent, err := budgetRepo.GetSpentByCategory(ctx, testUser, "PHP", from, to)
	if err != nil {
		t.Fatalf("get spent: %v", err)
	}
	if got := spent["Food"]; got != 1000 {
		t.Errorf("February Food spend = %d, want 1000 (March 1 excluded)", got)
	}
}

// TestCoordinatorRollbackLeavesNoOrphanRows pins the atomicity contract: when
// the coordinated function fails after writing, every row written inside the
// transaction is rolled back — no orphan transfer, no phantom contribution.
func TestCoordinatorRollbackLeavesNoOrphanRows(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	goalRepo := NewGoalRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)
	coordinator := NewCoordinator(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustWallet(t, walletRepo, "w-php2", "PHP", 0)
	goal, err := domain.NewSavingsGoal("goal-1", testUser, "Trip", 100000, nil, "w-php", "PHP")
	if err != nil {
		t.Fatalf("new goal: %v", err)
	}
	if err := goalRepo.SaveGoal(ctx, goal); err != nil {
		t.Fatalf("save goal: %v", err)
	}

	note := "contribution"
	contribution, err := domain.NewGoalContribution("c-1", "goal-1", 1000, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &note, nil, nil)
	if err != nil {
		t.Fatalf("new contribution: %v", err)
	}
	transfer, err := domain.NewWalletTransfer("tr-1", "tr-1", "w-php", "w-php2", "backing transfer", 1000, 1000, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new transfer: %v", err)
	}

	boom := errors.New("boom")
	err = coordinator.WithTx(ctx, func(txCtx context.Context) error {
		if err := goalRepo.SaveContribution(txCtx, contribution); err != nil {
			return err
		}
		if err := transferRepo.Save(txCtx, transfer); err != nil {
			return err
		}
		return boom
	})
	if err != boom {
		t.Fatalf("expected boom, got %v", err)
	}

	var contributions int
	if err := db.QueryRow("SELECT COUNT(*) FROM goal_contributions").Scan(&contributions); err != nil {
		t.Fatalf("count contributions: %v", err)
	}
	if contributions != 0 {
		t.Errorf("expected 0 contributions after rollback, got %d", contributions)
	}
	var transfers int
	if err := db.QueryRow("SELECT COUNT(*) FROM wallet_transfers").Scan(&transfers); err != nil {
		t.Fatalf("count transfers: %v", err)
	}
	if transfers != 0 {
		t.Errorf("expected 0 transfers after rollback, got %d", transfers)
	}
}

// TestTransferNeverLandsInTransactions pins the structural invariant: a
// transfer is a wallet-to-wallet movement and must never appear in the
// transactions table, so it can never be counted as income or expense.
func TestTransferNeverLandsInTransactions(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustWallet(t, walletRepo, "w-usd", "USD", 0)

	tr, err := domain.NewWalletTransfer("tr-1", "tr-1", "w-php", "w-usd", "convert", 10000, 5000, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new transfer: %v", err)
	}
	if err := transferRepo.Save(ctx, tr); err != nil {
		t.Fatalf("save transfer: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&count); err != nil {
		t.Fatalf("count transactions: %v", err)
	}
	if count != 0 {
		t.Errorf("transfer created %d transaction rows; transfers must never be income/expense", count)
	}
}

// TestExecutorUsesTransactionFromContext verifies the executor plumbing: a repo
// call inside Coordinator.WithTx runs against the transaction, and a call
// outside runs against the database handle.
func TestExecutorUsesTransactionFromContext(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	coordinator := NewCoordinator(db)

	// Outside a transaction, executor returns the fallback handle.
	if got := executor(ctx, db); got != db {
		t.Fatal("executor outside tx should return the fallback *sql.DB")
	}

	// Inside a transaction, executor returns the *sql.Tx.
	err := coordinator.WithTx(ctx, func(txCtx context.Context) error {
		tx, ok := txCtx.Value(txKey{}).(*sql.Tx)
		if !ok {
			t.Fatal("no *sql.Tx in context")
		}
		if got := executor(txCtx, db); got != tx {
			t.Fatal("executor inside tx should return the *sql.Tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}
}

// TestIsUniqueViolationRecognizesSQLiteConstraintError pins the string match
// that the services rely on to detect replay races.
func TestIsUniqueViolationRecognizesSQLiteConstraintError(t *testing.T) {
	if !isUniqueViolation(errors.New("UNIQUE constraint failed: transactions.idempotency_key")) {
		t.Fatal("expected UNIQUE constraint failure to be recognized")
	}
	if isUniqueViolation(errors.New("no such table: transactions")) {
		t.Fatal("unrelated error must not be treated as a unique violation")
	}
	if isUniqueViolation(nil) {
		t.Fatal("nil must not be a unique violation")
	}
}

// TestOptionalStringPinsNilForEmpty ensures empty strings are stored as NULL
// so the partial unique indexes ignore them.
func TestOptionalStringPinsNilForEmpty(t *testing.T) {
	if got := optionalString(""); got != nil {
		t.Fatalf("empty string should map to nil, got %v", got)
	}
	if got := optionalString("key"); got == nil || *got != "key" {
		t.Fatalf("non-empty string should round trip, got %v", got)
	}
}
