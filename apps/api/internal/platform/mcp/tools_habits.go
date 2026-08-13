package mcp

import (
	"context"
	"time"

	habitapp "github.com/jjspscl/my/internal/contexts/habits/application"
	"github.com/jjspscl/my/internal/platform/bootstrap"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type habitsListInput struct {
	Date time.Time `json:"date,omitempty"`
}

type completionsInput struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
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

func registerHabitsReadTools(server *mcpsdk.Server, app *bootstrap.App) {
	readOnly := &mcpsdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
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

func registerHabitsWriteTools(server *mcpsdk.Server, app *bootstrap.App) {
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_create", Description: "Create a habit.", Annotations: writable}, func(ctx context.Context, in createHabitInput) (any, error) {
		return app.Habit.Create(ctx, app.Cfg.UserEmail, habitapp.CreateHabitInput{Name: in.Name, Color: in.Color, Frequency: in.Frequency, TargetPerWeek: in.TargetPerWeek})
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_toggle", Description: "Toggle completion for a habit on a YYYY-MM-DD date; defaults to today.", Annotations: writable}, func(ctx context.Context, in toggleHabitInput) (any, error) {
		return app.Habit.ToggleCompletion(ctx, in.ID, app.Cfg.UserEmail, in.Date, nil)
	})
	registerTool(server, app.Log, &mcpsdk.Tool{Name: "habits_archive", Description: "Archive a habit. This action is destructive.", Annotations: destructive()}, func(ctx context.Context, in idInput) (any, error) {
		return nil, app.Habit.Archive(ctx, in.ID, app.Cfg.UserEmail)
	})
}
