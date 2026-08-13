package domain

import (
	"fmt"
	"strings"
	"time"
)

type GoalStatus string

const (
	GoalNotStarted GoalStatus = "not_started"
	GoalInProgress GoalStatus = "in_progress"
	GoalAchieved   GoalStatus = "achieved"
	GoalBehind     GoalStatus = "behind"

	MaxGoalNameLen = 200
)

type SavingsGoal struct {
	ID                string
	UserEmail         string
	Name              string
	TargetAmountCents int64
	TargetDate        *time.Time
	TargetWalletID    string
	Currency          string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type GoalContribution struct {
	ID             string
	GoalID         string
	AmountCents    int64
	ContributedAt  time.Time
	Note           *string
	SourceWalletID *string
	TransferID     *string
	IdempotencyKey string
	CreatedAt      time.Time
}

type GoalSummary struct {
	Goal                 SavingsGoal
	CurrentAmountCents   int64
	RemainingAmountCents int64
	ProgressPercent      int
	RequiredMonthlyCents *int64
	Status               GoalStatus
}

// NewSavingsGoal validates and creates a SavingsGoal.
func NewSavingsGoal(id, userEmail, name string, targetAmountCents int64, targetDate *time.Time, targetWalletID, currency string) (*SavingsGoal, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > MaxGoalNameLen {
		return nil, fmt.Errorf("name too long (max %d)", MaxGoalNameLen)
	}
	if targetAmountCents <= 0 {
		return nil, fmt.Errorf("target amount must be positive")
	}
	if targetWalletID == "" {
		return nil, fmt.Errorf("target wallet is required")
	}
	if currency == "" {
		currency = "PHP"
	}

	return &SavingsGoal{
		ID:                id,
		UserEmail:         userEmail,
		Name:              strings.TrimSpace(name),
		TargetAmountCents: targetAmountCents,
		TargetDate:        targetDate,
		TargetWalletID:    targetWalletID,
		Currency:          currency,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}

// NewGoalContribution validates and creates a GoalContribution.
func NewGoalContribution(id, goalID string, amountCents int64, contributedAt time.Time, note *string, sourceWalletID, transferID *string) (*GoalContribution, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if goalID == "" {
		return nil, fmt.Errorf("goal id is required")
	}
	if amountCents <= 0 {
		return nil, fmt.Errorf("contribution amount must be positive")
	}
	if contributedAt.IsZero() {
		return nil, fmt.Errorf("contributed at is required")
	}

	return &GoalContribution{
		ID:             id,
		GoalID:         goalID,
		AmountCents:    amountCents,
		ContributedAt:  contributedAt,
		Note:           note,
		SourceWalletID: sourceWalletID,
		TransferID:     transferID,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// ComputeGoalSummary computes a summary for a goal given its contributions
// total. now is the current instant in the user's financial timezone; it is a
// parameter so callers control the calendar, not the UTC clock.
func ComputeGoalSummary(goal *SavingsGoal, currentAmountCents int64, now time.Time) GoalSummary {
	remaining := goal.TargetAmountCents - currentAmountCents
	if remaining < 0 {
		remaining = 0
	}

	progress := 0
	if goal.TargetAmountCents > 0 {
		p := int((currentAmountCents * 100) / goal.TargetAmountCents)
		if p > 100 {
			p = 100
		}
		progress = p
	}

	var monthly *int64
	if goal.TargetDate != nil && remaining > 0 {
		if goal.TargetDate.After(now) {
			monthsLeft := monthsBetween(now, *goal.TargetDate)
			if monthsLeft > 0 {
				m := (remaining + int64(monthsLeft) - 1) / int64(monthsLeft) // ceil division
				monthly = &m
			}
		}
	}

	status := GoalInProgress
	if currentAmountCents == 0 {
		status = GoalNotStarted
	} else if currentAmountCents >= goal.TargetAmountCents {
		status = GoalAchieved
	} else if goal.TargetDate != nil && now.After(*goal.TargetDate) {
		status = GoalBehind
	}

	return GoalSummary{
		Goal:                 *goal,
		CurrentAmountCents:   currentAmountCents,
		RemainingAmountCents: remaining,
		ProgressPercent:      progress,
		RequiredMonthlyCents: monthly,
		Status:               status,
	}
}

func monthsBetween(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	months := (to.Year()-from.Year())*12 + int(to.Month()) - int(from.Month())
	if to.Day() < from.Day() {
		months--
	}
	if months < 1 {
		months = 1
	}
	return months
}
