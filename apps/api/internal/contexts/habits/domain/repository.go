package domain

import (
	"context"
	"time"
)

type HabitRepository interface {
	SaveHabit(ctx context.Context, h *Habit) error
	ListActive(ctx context.Context, userEmail string) ([]*Habit, error)
	FindByID(ctx context.Context, id, userEmail string) (*Habit, error)
	Archive(ctx context.Context, id, userEmail string) error

	SaveCompletion(ctx context.Context, c *HabitCompletion) error
	DeleteCompletion(ctx context.Context, habitID string, date time.Time) error
	GetCompletion(ctx context.Context, habitID string, date time.Time) (*HabitCompletion, error)
	GetCompletionsInRange(ctx context.Context, habitID string, from, to time.Time) ([]*HabitCompletion, error)
	GetAllCompletionsInRange(ctx context.Context, userEmail string, from, to time.Time) (map[string][]*HabitCompletion, error)
}
