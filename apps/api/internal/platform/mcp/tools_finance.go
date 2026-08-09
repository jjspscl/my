package mcp

import (
	"context"
	"time"

	financeapp "github.com/jjspscl/my/internal/contexts/finance/application"
	financedomain "github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/bootstrap"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type emptyInput struct{}

type listTransactionsInput struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Limit  int       `json:"limit,omitempty"`
	Offset int       `json:"offset,omitempty"`
}

type todayTotalInput struct{}

type budgetSummaryInput struct {
	Month string `json:"month"`
}

type upcomingBillsInput struct {
	DaysAhead int `json:"days_ahead,omitempty"`
}

type listTransfersInput struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

type createTransactionInput struct {
	AmountCents     int64                         `json:"amount_cents"`
	Category        string                        `json:"category"`
	Description     string                        `json:"description,omitempty"`
	Type            financedomain.TransactionType `json:"type"`
	WalletID        string                        `json:"wallet_id,omitempty"`
	TransactionDate time.Time                     `json:"transaction_date"`
	IdempotencyKey  string                        `json:"idempotency_key,omitempty"`
}

type upsertBudgetInput struct {
	Month      string                           `json:"month"`
	Categories []financeapp.BudgetCategoryInput `json:"categories"`
}

type createBillInput struct {
	Name         string                  `json:"name"`
	Category     string                  `json:"category"`
	AmountCents  int64                   `json:"amount_cents"`
	Currency     string                  `json:"currency,omitempty"`
	Frequency    financedomain.Frequency `json:"frequency"`
	DayOfMonth   int                     `json:"day_of_month"`
	StartDate    time.Time               `json:"start_date"`
	EndDate      *time.Time              `json:"end_date,omitempty"`
	AutoMatch    bool                    `json:"auto_match"`
	MatchPattern *string                 `json:"match_pattern,omitempty"`
}

type updateBillInput struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Category     string                  `json:"category"`
	AmountCents  int64                   `json:"amount_cents"`
	Currency     string                  `json:"currency,omitempty"`
	Frequency    financedomain.Frequency `json:"frequency"`
	DayOfMonth   int                     `json:"day_of_month"`
	StartDate    time.Time               `json:"start_date"`
	EndDate      *time.Time              `json:"end_date,omitempty"`
	AutoMatch    bool                    `json:"auto_match"`
	MatchPattern *string                 `json:"match_pattern,omitempty"`
}

type payBillInput struct {
	BillID            string    `json:"bill_id"`
	DueDate           time.Time `json:"due_date"`
	TransactionID     *string   `json:"transaction_id,omitempty"`
	CreateTransaction bool      `json:"create_transaction,omitempty"`
	WalletID          string    `json:"wallet_id,omitempty"`
}

type createGoalInput struct {
	Name              string     `json:"name"`
	TargetAmountCents int64      `json:"target_amount_cents"`
	TargetDate        *time.Time `json:"target_date,omitempty"`
	TargetWalletID    string     `json:"target_wallet_id"`
}

type updateGoalInput struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	TargetAmountCents int64      `json:"target_amount_cents"`
	TargetDate        *time.Time `json:"target_date,omitempty"`
	TargetWalletID    string     `json:"target_wallet_id"`
}

type goalContributionInput struct {
	GoalID          string    `json:"goal_id"`
	AmountCents     int64     `json:"amount_cents"`
	ContributedAt   time.Time `json:"contributed_at"`
	Note            *string   `json:"note,omitempty"`
	SourceWalletID  *string   `json:"source_wallet_id,omitempty"`
	FromAmountCents *int64    `json:"from_amount_cents,omitempty"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
}

type createWalletInput struct {
	Name                string                   `json:"name"`
	Kind                financedomain.WalletKind `json:"kind"`
	OpeningBalanceCents int64                    `json:"opening_balance_cents"`
}

type updateWalletInput struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Kind                financedomain.WalletKind `json:"kind"`
	OpeningBalanceCents int64                    `json:"opening_balance_cents"`
}

type createTransferInput struct {
	FromWalletID    string    `json:"from_wallet_id"`
	ToWalletID      string    `json:"to_wallet_id"`
	AmountCents     int64     `json:"amount_cents"`
	FromAmountCents *int64    `json:"from_amount_cents,omitempty"`
	ToAmountCents   *int64    `json:"to_amount_cents,omitempty"`
	Description     string    `json:"description,omitempty"`
	TransferDate    time.Time `json:"transfer_date"`
	IdempotencyKey  string    `json:"idempotency_key,omitempty"`
}

func registerFinanceReadTools(server *mcpsdk.Server, app *bootstrap.App) {
	readOnly := &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_transactions", Description: "List transactions in a UTC date range.", Annotations: readOnly}, func(ctx context.Context, in listTransactionsInput) (any, error) {
		return app.Tx.List(ctx, app.Cfg.UserEmail, financeapp.TransactionFilter{From: in.From, To: in.To, Limit: in.Limit, Offset: in.Offset})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_today_total", Description: "Return today's income, expense, and net total in the default currency.", Annotations: readOnly}, func(ctx context.Context, _ todayTotalInput) (any, error) {
		return app.Tx.GetTodayTotal(ctx, app.Cfg.UserEmail, app.Cfg.DefaultCurrency)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_budget_summary", Description: "Return allocated, spent, and remaining budget by category for a month.", Annotations: readOnly}, func(ctx context.Context, in budgetSummaryInput) (any, error) {
		return app.Budget.GetSummary(ctx, app.Cfg.UserEmail, in.Month)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_bills", Description: "List recurring bills.", Annotations: readOnly}, func(ctx context.Context, _ emptyInput) (any, error) {
		return app.Bill.List(ctx, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_upcoming_bills", Description: "List bill occurrences and payment status through requested days ahead.", Annotations: readOnly}, func(ctx context.Context, in upcomingBillsInput) (any, error) {
		if in.DaysAhead == 0 {
			in.DaysAhead = 14
		}
		return app.Bill.GetUpcoming(ctx, app.Cfg.UserEmail, in.DaysAhead)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_goals", Description: "List savings goals with current progress summaries.", Annotations: readOnly}, func(ctx context.Context, _ emptyInput) (any, error) {
		return app.Goal.ListSummaries(ctx, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_wallets", Description: "List wallets with current balances.", Annotations: readOnly}, func(ctx context.Context, _ emptyInput) (any, error) {
		return app.Wallet.ListWithBalances(ctx, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_transfers", Description: "List wallet transfers.", Annotations: readOnly}, func(ctx context.Context, in listTransfersInput) (any, error) {
		return app.Transfer.List(ctx, app.Cfg.UserEmail, financeapp.TransferFilter{Limit: in.Limit, Offset: in.Offset})
	})
}

func registerFinanceWriteTools(server *mcpsdk.Server, app *bootstrap.App) {
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_transaction", Description: "Create an income or expense transaction. Amount is positive minor currency units. Idempotency key makes retries safe.", Annotations: writable}, func(ctx context.Context, in createTransactionInput) (any, error) {
		return app.Tx.Create(ctx, app.Cfg.UserEmail, financeapp.CreateTransactionInput{AmountCents: in.AmountCents, Category: in.Category, Description: in.Description, Type: in.Type, WalletID: in.WalletID, TransactionDate: in.TransactionDate, IdempotencyKey: in.IdempotencyKey})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_delete_transaction", Description: "Permanently delete a transaction. This action is irreversible.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Tx.Delete(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_upsert_budget", Description: "Create or replace a month's budget categories.", Annotations: writable}, func(ctx context.Context, in upsertBudgetInput) (any, error) {
		return app.Budget.UpsertBudget(ctx, app.Cfg.UserEmail, financeapp.UpsertBudgetInput{Month: in.Month, Categories: in.Categories})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_bill", Description: "Create a recurring bill. Currency defaults to the dashboard default when omitted.", Annotations: writable}, func(ctx context.Context, in createBillInput) (any, error) {
		return app.Bill.Create(ctx, app.Cfg.UserEmail, financeapp.CreateBillInput{Name: in.Name, Category: in.Category, AmountCents: in.AmountCents, Currency: in.Currency, Frequency: in.Frequency, DayOfMonth: in.DayOfMonth, StartDate: in.StartDate, EndDate: in.EndDate, AutoMatch: in.AutoMatch, MatchPattern: in.MatchPattern})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_update_bill", Description: "Update an existing recurring bill. Empty currency keeps the current value.", Annotations: writable}, func(ctx context.Context, in updateBillInput) (any, error) {
		return app.Bill.Update(ctx, app.Cfg.UserEmail, financeapp.UpdateBillInput{ID: in.ID, Name: in.Name, Category: in.Category, AmountCents: in.AmountCents, Currency: in.Currency, Frequency: in.Frequency, DayOfMonth: in.DayOfMonth, StartDate: in.StartDate, EndDate: in.EndDate, AutoMatch: in.AutoMatch, MatchPattern: in.MatchPattern})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_delete_bill", Description: "Permanently delete a recurring bill. This action is irreversible.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Bill.Delete(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_pay_bill", Description: "Mark a bill occurrence as paid. Writes a payment record; rerunning for the same due date is safe. Set create_transaction to also book the expense transaction (atomically) and link it; wallet_id selects the wallet, defaulting to the user's default wallet.", Annotations: writable}, func(ctx context.Context, in payBillInput) (any, error) {
		return app.Bill.MarkPaid(ctx, app.Cfg.UserEmail, financeapp.MarkPaidInput{BillID: in.BillID, DueDate: in.DueDate, TransactionID: in.TransactionID, CreateTransaction: in.CreateTransaction, WalletID: in.WalletID})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_goal", Description: "Create a savings goal.", Annotations: writable}, func(ctx context.Context, in createGoalInput) (any, error) {
		return app.Goal.Create(ctx, app.Cfg.UserEmail, financeapp.CreateGoalInput{Name: in.Name, TargetAmountCents: in.TargetAmountCents, TargetDate: in.TargetDate, TargetWalletID: in.TargetWalletID})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_update_goal", Description: "Update a savings goal.", Annotations: writable}, func(ctx context.Context, in updateGoalInput) (any, error) {
		return app.Goal.Update(ctx, app.Cfg.UserEmail, financeapp.UpdateGoalInput{ID: in.ID, Name: in.Name, TargetAmountCents: in.TargetAmountCents, TargetDate: in.TargetDate, TargetWalletID: in.TargetWalletID})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_delete_goal", Description: "Permanently delete a savings goal. This action is irreversible.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Goal.Delete(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_add_goal_contribution", Description: "Add money to a savings goal, optionally creating a wallet transfer. Idempotency key makes retries safe.", Annotations: writable}, func(ctx context.Context, in goalContributionInput) (any, error) {
		return app.Goal.AddContribution(ctx, app.Cfg.UserEmail, financeapp.AddContributionInput{GoalID: in.GoalID, AmountCents: in.AmountCents, ContributedAt: in.ContributedAt, Note: in.Note, SourceWalletID: in.SourceWalletID, FromAmountCents: in.FromAmountCents, IdempotencyKey: in.IdempotencyKey})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_wallet", Description: "Create a wallet. Amount is opening balance in minor currency units.", Annotations: writable}, func(ctx context.Context, in createWalletInput) (any, error) {
		return app.Wallet.Create(ctx, app.Cfg.UserEmail, app.Cfg.DefaultCurrency, financeapp.CreateWalletInput{Name: in.Name, Kind: in.Kind, OpeningBalanceCents: in.OpeningBalanceCents})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_update_wallet", Description: "Update a wallet's name, kind, or opening balance.", Annotations: writable}, func(ctx context.Context, in updateWalletInput) (any, error) {
		return app.Wallet.Update(ctx, app.Cfg.UserEmail, financeapp.UpdateWalletInput{ID: in.ID, Name: in.Name, Kind: in.Kind, OpeningBalanceCents: in.OpeningBalanceCents})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_archive_wallet", Description: "Archive a wallet. This action is destructive and cannot be undone through MCP.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Wallet.Archive(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_transfer", Description: "Transfer money between two wallets. For same-currency wallets amount_cents is used for both legs; cross-currency transfers must supply from_amount_cents and to_amount_cents. Supply idempotency_key and reuse it across retries so a retried transfer is not applied twice.", Annotations: writable}, func(ctx context.Context, in createTransferInput) (any, error) {
		fromAmount := in.AmountCents
		if in.FromAmountCents != nil {
			fromAmount = *in.FromAmountCents
		}
		toAmount := in.AmountCents
		if in.ToAmountCents != nil {
			toAmount = *in.ToAmountCents
		}
		return app.Transfer.Create(ctx, app.Cfg.UserEmail, financeapp.CreateTransferInput{FromWalletID: in.FromWalletID, ToWalletID: in.ToWalletID, FromAmountCents: fromAmount, ToAmountCents: toAmount, Description: in.Description, TransferDate: in.TransferDate, IdempotencyKey: in.IdempotencyKey})
	})
}
