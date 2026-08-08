package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	financeapp "github.com/jjspscl/my/internal/contexts/finance/application"
	financedomain "github.com/jjspscl/my/internal/contexts/finance/domain"
	habitapp "github.com/jjspscl/my/internal/contexts/habits/application"
	"github.com/jjspscl/my/internal/platform/bootstrap"
	platformversion "github.com/jjspscl/my/internal/platform/version"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Options struct {
	ReadOnly bool
}

func NewServer(app *bootstrap.App, opts Options) *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "my",
		Version: appVersion(app),
	}, &mcpsdk.ServerOptions{
		Logger: app.Log,
		Instructions: "Personal dashboard for finance, habits, and daily life. " +
			"All data belongs to the configured single user. Confirm destructive actions before calling them.",
	})

	registerReadTools(server, app)
	if !opts.ReadOnly {
		registerWriteTools(server, app)
	}
	registerResources(server, app)
	registerPrompts(server)
	return server
}

func appVersion(app *bootstrap.App) string {
	return platformversion.Version
}

type toolHandler[In any] func(context.Context, In) (any, error)

func registerTool[In any](server *mcpsdk.Server, logger *slog.Logger, tool *mcpsdk.Tool, handler toolHandler[In]) {
	mcpsdk.AddTool[In, any](server, tool, func(ctx context.Context, _ *mcpsdk.CallToolRequest, input In) (_ *mcpsdk.CallToolResult, output any, err error) {
		started := time.Now()
		defer func() {
			if logger != nil {
				attrs := []any{
					slog.String("tool", tool.Name),
					slog.Duration("duration", time.Since(started)),
					slog.String("outcome", outcome(err)),
				}
				if err != nil {
					attrs = append(attrs, slog.String("error", err.Error()))
				}
				logger.Info("mcp tool call", attrs...)
			}
		}()
		output, err = handler(ctx, input)
		return nil, output, err
	})
}

func outcome(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

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

type habitsListInput struct {
	Date time.Time `json:"date,omitempty"`
}

type completionsInput struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func registerReadTools(server *mcpsdk.Server, app *bootstrap.App) {
	readOnly := &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_list_transactions", Description: "List transactions in a UTC date range.", Annotations: readOnly}, func(ctx context.Context, in listTransactionsInput) (any, error) {
		return app.Tx.List(ctx, app.Cfg.UserEmail, financeapp.TransactionFilter{From: in.From, To: in.To, Limit: in.Limit, Offset: in.Offset})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_today_total", Description: "Return today's income, expense, and net total.", Annotations: readOnly}, func(ctx context.Context, _ todayTotalInput) (any, error) {
		return app.Tx.GetTodayTotal(ctx, app.Cfg.UserEmail)
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
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_list", Description: "List active habits with completion status and streak for a UTC date.", Annotations: readOnly}, func(ctx context.Context, in habitsListInput) (any, error) {
		if in.Date.IsZero() {
			in.Date = time.Now().UTC()
		}
		return app.Habit.ListWithStatus(ctx, app.Cfg.UserEmail, in.Date)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_completions", Description: "Return grouped habit completions for a UTC date range.", Annotations: readOnly}, func(ctx context.Context, in completionsInput) (any, error) {
		return app.Habit.GetAllCompletionsGrouped(ctx, app.Cfg.UserEmail, in.From, in.To)
	})
}

type createTransactionInput struct {
	AmountCents     int64                         `json:"amount_cents"`
	Category        string                        `json:"category"`
	Description     string                        `json:"description,omitempty"`
	Type            financedomain.TransactionType `json:"type"`
	WalletID        string                        `json:"wallet_id,omitempty"`
	TransactionDate time.Time                     `json:"transaction_date"`
}

type idInput struct {
	ID string `json:"id"`
}

type upsertBudgetInput struct {
	Month      string                           `json:"month"`
	Categories []financeapp.BudgetCategoryInput `json:"categories"`
}

type createBillInput struct {
	Name         string                  `json:"name"`
	Category     string                  `json:"category"`
	AmountCents  int64                   `json:"amount_cents"`
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
	Frequency    financedomain.Frequency `json:"frequency"`
	DayOfMonth   int                     `json:"day_of_month"`
	StartDate    time.Time               `json:"start_date"`
	EndDate      *time.Time              `json:"end_date,omitempty"`
	AutoMatch    bool                    `json:"auto_match"`
	MatchPattern *string                 `json:"match_pattern,omitempty"`
}

type payBillInput struct {
	BillID        string    `json:"bill_id"`
	DueDate       time.Time `json:"due_date"`
	TransactionID *string   `json:"transaction_id,omitempty"`
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
	GoalID         string    `json:"goal_id"`
	AmountCents    int64     `json:"amount_cents"`
	ContributedAt  time.Time `json:"contributed_at"`
	Note           *string   `json:"note,omitempty"`
	SourceWalletID *string   `json:"source_wallet_id,omitempty"`
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
	FromWalletID string    `json:"from_wallet_id"`
	ToWalletID   string    `json:"to_wallet_id"`
	AmountCents  int64     `json:"amount_cents"`
	Description  string    `json:"description,omitempty"`
	TransferDate time.Time `json:"transfer_date"`
}

type createHabitInput struct {
	Name          string `json:"name"`
	Color         string `json:"color"`
	Frequency     string `json:"frequency"`
	TargetPerWeek int    `json:"target_per_week"`
}

type toggleHabitInput struct {
	ID   string `json:"id"`
	Date string `json:"date,omitempty"`
}

func registerWriteTools(server *mcpsdk.Server, app *bootstrap.App) {
	destructive := func() *mcpsdk.ToolAnnotations {
		return &mcpsdk.ToolAnnotations{DestructiveHint: boolPointer(true), IdempotentHint: false}
	}
	writable := &mcpsdk.ToolAnnotations{DestructiveHint: boolPointer(false)}

	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_transaction", Description: "Create an income or expense transaction. Amount is positive minor currency units.", Annotations: writable}, func(ctx context.Context, in createTransactionInput) (any, error) {
		return app.Tx.Create(ctx, app.Cfg.UserEmail, financeapp.CreateTransactionInput{AmountCents: in.AmountCents, Category: in.Category, Description: in.Description, Type: in.Type, WalletID: in.WalletID, TransactionDate: in.TransactionDate})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_delete_transaction", Description: "Permanently delete a transaction. This action is irreversible.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Tx.Delete(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_upsert_budget", Description: "Create or replace a month's budget categories.", Annotations: writable}, func(ctx context.Context, in upsertBudgetInput) (any, error) {
		return app.Budget.UpsertBudget(ctx, app.Cfg.UserEmail, financeapp.UpsertBudgetInput{Month: in.Month, Categories: in.Categories})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_bill", Description: "Create a recurring bill.", Annotations: writable}, func(ctx context.Context, in createBillInput) (any, error) {
		return app.Bill.Create(ctx, app.Cfg.UserEmail, financeapp.CreateBillInput{Name: in.Name, Category: in.Category, AmountCents: in.AmountCents, Frequency: in.Frequency, DayOfMonth: in.DayOfMonth, StartDate: in.StartDate, EndDate: in.EndDate, AutoMatch: in.AutoMatch, MatchPattern: in.MatchPattern})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_update_bill", Description: "Update an existing recurring bill.", Annotations: writable}, func(ctx context.Context, in updateBillInput) (any, error) {
		return app.Bill.Update(ctx, app.Cfg.UserEmail, financeapp.UpdateBillInput{ID: in.ID, Name: in.Name, Category: in.Category, AmountCents: in.AmountCents, Frequency: in.Frequency, DayOfMonth: in.DayOfMonth, StartDate: in.StartDate, EndDate: in.EndDate, AutoMatch: in.AutoMatch, MatchPattern: in.MatchPattern})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_delete_bill", Description: "Permanently delete a recurring bill. This action is irreversible.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Bill.Delete(ctx, in.ID, app.Cfg.UserEmail)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_pay_bill", Description: "Mark a bill occurrence paid. This writes a payment record and is irreversible through MCP.", Annotations: destructive()}, func(ctx context.Context, in payBillInput) (any, error) {
		return app.Bill.MarkPaid(ctx, in.BillID, app.Cfg.UserEmail, in.DueDate, in.TransactionID)
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
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_add_goal_contribution", Description: "Add money to a savings goal, optionally creating a wallet transfer.", Annotations: writable}, func(ctx context.Context, in goalContributionInput) (any, error) {
		return app.Goal.AddContribution(ctx, app.Cfg.UserEmail, financeapp.AddContributionInput{GoalID: in.GoalID, AmountCents: in.AmountCents, ContributedAt: in.ContributedAt, Note: in.Note, SourceWalletID: in.SourceWalletID})
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
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "finance_create_transfer", Description: "Transfer money between two wallets.", Annotations: writable}, func(ctx context.Context, in createTransferInput) (any, error) {
		return app.Transfer.Create(ctx, app.Cfg.UserEmail, financeapp.CreateTransferInput{FromWalletID: in.FromWalletID, ToWalletID: in.ToWalletID, AmountCents: in.AmountCents, Description: in.Description, TransferDate: in.TransferDate})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_create", Description: "Create a habit.", Annotations: writable}, func(ctx context.Context, in createHabitInput) (any, error) {
		return app.Habit.Create(ctx, app.Cfg.UserEmail, habitapp.CreateHabitInput{Name: in.Name, Color: in.Color, Frequency: in.Frequency, TargetPerWeek: in.TargetPerWeek})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_toggle", Description: "Toggle completion for a habit on a YYYY-MM-DD date; defaults to today.", Annotations: writable}, func(ctx context.Context, in toggleHabitInput) (any, error) {
		return app.Habit.ToggleCompletion(ctx, in.ID, app.Cfg.UserEmail, in.Date)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_archive", Description: "Archive a habit. This action is destructive.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Habit.Archive(ctx, in.ID, app.Cfg.UserEmail)
	})
}

func boolPointer(value bool) *bool {
	return &value
}

func registerResources(server *mcpsdk.Server, app *bootstrap.App) {
	resources := []struct {
		name, uri, description string
		handler                mcpsdk.ResourceHandler
	}{
		{"wallets", "my://wallets", "Wallets with current balances.", func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			value, err := app.Wallet.ListWithBalances(ctx, app.Cfg.UserEmail)
			return resourceJSON("my://wallets", value, err)
		}},
		{"budget-current", "my://budget/current", "Current month's budget summary.", func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			value, err := app.Budget.GetSummary(ctx, app.Cfg.UserEmail, time.Now().UTC().Format("2006-01"))
			return resourceJSON("my://budget/current", value, err)
		}},
		{"bills-upcoming", "my://bills/upcoming", "Upcoming bills for the next fourteen days.", func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			value, err := app.Bill.GetUpcoming(ctx, app.Cfg.UserEmail, 14)
			return resourceJSON("my://bills/upcoming", value, err)
		}},
		{"habits-today", "my://habits/today", "Today's habits with completion status.", func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			value, err := app.Habit.ListWithStatus(ctx, app.Cfg.UserEmail, time.Now().UTC())
			return resourceJSON("my://habits/today", value, err)
		}},
	}
	for _, resource := range resources {
		server.AddResource(&mcpsdk.Resource{Name: resource.name, URI: resource.uri, Description: resource.description, MIMEType: "application/json"}, resource.handler)
	}

	server.AddResource(&mcpsdk.Resource{Name: "dashboard-snapshot", URI: "my://dashboard/snapshot", Description: "Composite daily dashboard snapshot.", MIMEType: "application/json"}, func(ctx context.Context, _ *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		total, err := app.Tx.GetTodayTotal(ctx, app.Cfg.UserEmail)
		if err != nil {
			return nil, err
		}
		bills, err := app.Bill.GetUpcoming(ctx, app.Cfg.UserEmail, 14)
		if err != nil {
			return nil, err
		}
		habits, err := app.Habit.ListWithStatus(ctx, app.Cfg.UserEmail, time.Now().UTC())
		if err != nil {
			return nil, err
		}
		completed := 0
		for _, habit := range habits {
			if habit.CompletedToday {
				completed++
			}
		}
		return resourceJSON("my://dashboard/snapshot", map[string]any{
			"today_total":            total,
			"upcoming_bills_count":   len(bills),
			"habits_completed":       completed,
			"habits_total":           len(habits),
			"habit_completion_ratio": ratio(completed, len(habits)),
		}, nil)
	})
}

func ratio(done, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(done) / float64(total)
}

func resourceJSON(uri string, value any, err error) (*mcpsdk.ReadResourceResult, error) {
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}
	return &mcpsdk.ReadResourceResult{Contents: []*mcpsdk.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(data)}}}, nil
}

func registerPrompts(server *mcpsdk.Server) {
	server.AddPrompt(&mcpsdk.Prompt{Name: "weekly_finance_review", Description: "Review recent spending, budget health, bills, and goals."}, staticPrompt("Call finance_list_transactions, finance_budget_summary, finance_upcoming_bills, and finance_list_goals. Summarize trends and concrete next actions."))
	server.AddPrompt(&mcpsdk.Prompt{Name: "budget_health_check", Description: "Check budget health for a month.", Arguments: []*mcpsdk.PromptArgument{{Name: "month", Description: "Month in YYYY-MM format.", Required: false}}}, staticPrompt("Use finance_budget_summary for the requested month, then identify categories with negative remaining balance and suggest changes."))
	server.AddPrompt(&mcpsdk.Prompt{Name: "habit_streak_report", Description: "Review habit completion and streak performance.", Arguments: []*mcpsdk.PromptArgument{{Name: "days", Description: "Number of days to review.", Required: false}}}, staticPrompt("Use habits_completions and habits_list. Report completion patterns, current streaks, and one practical improvement per weak habit."))
}

func staticPrompt(text string) mcpsdk.PromptHandler {
	return func(context.Context, *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		return &mcpsdk.GetPromptResult{Messages: []*mcpsdk.PromptMessage{{Role: "user", Content: &mcpsdk.TextContent{Text: text}}}}, nil
	}
}
