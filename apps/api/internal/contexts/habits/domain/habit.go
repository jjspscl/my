package domain

import "time"

type Frequency string

const (
	FrequencyDaily  Frequency = "daily"
	FrequencyWeekly Frequency = "weekly"
)

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
