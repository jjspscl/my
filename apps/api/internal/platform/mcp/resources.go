package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/platform/bootstrap"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

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
			value, err := app.Budget.GetSummary(ctx, app.Cfg.UserEmail, time.Now().In(app.Cfg.Location).Format("2006-01"))
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
		total, err := app.Tx.GetTodayTotal(ctx, app.Cfg.UserEmail, app.Cfg.DefaultCurrency)
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
