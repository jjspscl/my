package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(server *mcpsdk.Server) {
	server.AddPrompt(&mcpsdk.Prompt{Name: "weekly_finance_review", Description: "Review recent spending, budget health, bills, and goals."}, staticPrompt("Call finance_cash_flow_summary, finance_spending_summary, finance_budget_health, finance_goal_health, and finance_upcoming_bills. Summarize trends, flag risks, and propose concrete next actions."))
	server.AddPrompt(&mcpsdk.Prompt{Name: "budget_health_check", Description: "Check budget health for a month.", Arguments: []*mcpsdk.PromptArgument{{Name: "month", Description: "Month in YYYY-MM format.", Required: false}}}, staticPrompt("Use finance_budget_health for the requested month, then identify categories at risk of overspending and suggest changes."))
	server.AddPrompt(&mcpsdk.Prompt{Name: "habit_streak_report", Description: "Review habit completion and streak performance.", Arguments: []*mcpsdk.PromptArgument{{Name: "days", Description: "Number of days to review.", Required: false}}}, staticPrompt("Use habits_completions and habits_list. Report completion patterns, current streaks, and one practical improvement per weak habit."))
}

func staticPrompt(text string) mcpsdk.PromptHandler {
	return func(context.Context, *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		return &mcpsdk.GetPromptResult{Messages: []*mcpsdk.PromptMessage{{Role: "user", Content: &mcpsdk.TextContent{Text: text}}}}, nil
	}
}
