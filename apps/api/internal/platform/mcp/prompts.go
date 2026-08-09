package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

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
