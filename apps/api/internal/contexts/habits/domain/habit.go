package domain

import (
	"fmt"
	"time"
)

type Frequency string

const (
	FrequencyDaily  Frequency = "daily"
	FrequencyWeekly Frequency = "weekly"
)

// ValidPaletteTokens matches frontend palette tokens.
var ValidPaletteTokens = map[string]bool{
	"red": true, "orange": true, "amber": true, "yellow": true,
	"green": true, "teal": true, "cyan": true, "blue": true,
	"indigo": true, "purple": true, "pink": true, "slate": true,
}

type Habit struct {
	ID            string
	UserEmail     string
	Name          string
	Color         string
	Frequency     Frequency
	TargetPerWeek int
	Archived      bool
	CreatedAt     time.Time
}

type HabitCompletion struct {
	ID            string
	HabitID       string
	CompletedDate time.Time
	CreatedAt     time.Time
}

type HabitWithStatus struct {
	Habit
	CompletedToday bool
	CurrentStreak  int
}

type HabitStreak struct {
	HabitID     string
	Name        string
	Color       string
	Frequency   Frequency
	StreakCount int
}

// NewHabit creates a validated habit. Returns error on invalid invariants.
func NewHabit(id, userEmail, name, color string, freq Frequency, targetPerWeek int) (*Habit, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return nil, fmt.Errorf("name too long (max 200)")
	}
	if color == "" {
		color = "blue"
	}
	if !ValidPaletteTokens[color] {
		return nil, fmt.Errorf("invalid color: %s", color)
	}
	if freq != FrequencyDaily && freq != FrequencyWeekly {
		freq = FrequencyDaily
	}
	if targetPerWeek < 1 {
		targetPerWeek = 1
	}
	if targetPerWeek > 7 {
		targetPerWeek = 7
	}

	return &Habit{
		ID:            id,
		UserEmail:     userEmail,
		Name:          name,
		Color:         color,
		Frequency:     freq,
		TargetPerWeek: targetPerWeek,
		Archived:      false,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
