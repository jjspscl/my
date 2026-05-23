package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/habits/domain"
)

type HabitService struct {
	repo domain.HabitRepository
}

func NewHabitService(repo domain.HabitRepository) *HabitService {
	return &HabitService{repo: repo}
}

type CreateHabitInput struct {
	Name          string
	Color         string
	Frequency     string
	TargetPerWeek int
}

func (s *HabitService) Create(ctx context.Context, userEmail string, input CreateHabitInput) (*domain.Habit, error) {
	h, err := domain.NewHabit(
		uuid.New().String(),
		userEmail,
		input.Name,
		input.Color,
		domain.Frequency(input.Frequency),
		input.TargetPerWeek,
	)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveHabit(ctx, h); err != nil {
		return nil, fmt.Errorf("save habit: %w", err)
	}
	return h, nil
}

type HabitWithStatusResult struct {
	Habit          domain.Habit `json:"-"`
	CompletedToday bool
	CurrentStreak  int
}

func (s *HabitService) ListWithStatus(ctx context.Context, userEmail string, date time.Time) ([]domain.HabitWithStatus, error) {
	habits, err := s.repo.ListActive(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}

	// Get all completions for the date range needed for streak calculation
	// Streak only looks back, so fetch from 90 days ago to today
	from := date.AddDate(0, 0, -90)
	compsByHabit, err := s.repo.GetAllCompletionsInRange(ctx, userEmail, from, date)
	if err != nil {
		return nil, fmt.Errorf("get completions: %w", err)
	}

	results := make([]domain.HabitWithStatus, 0, len(habits))
	for _, h := range habits {
		comps := compsByHabit[h.ID]
		completedToday := false
		for _, c := range comps {
			if c.CompletedDate.Format("2006-01-02") == date.Format("2006-01-02") {
				completedToday = true
				break
			}
		}
		streak := calculateStreak(h, comps, date)

		results = append(results, domain.HabitWithStatus{
			Habit:          *h,
			CompletedToday: completedToday,
			CurrentStreak:  streak,
		})
	}

	return results, nil
}

func calculateStreak(h *domain.Habit, comps []*domain.HabitCompletion, asOf time.Time) int {
	if len(comps) == 0 {
		return 0
	}

	// Build set of completed date strings
	completed := make(map[string]bool)
	for _, c := range comps {
		completed[c.CompletedDate.Format("2006-01-02")] = true
	}

	switch h.Frequency {
	case domain.FrequencyDaily:
		return dailyStreak(completed, asOf)
	case domain.FrequencyWeekly:
		return weeklyStreak(completed, asOf, h.TargetPerWeek)
	default:
		return 0
	}
}

func dailyStreak(completed map[string]bool, asOf time.Time) int {
	streak := 0
	cur := asOf

	// Allow today or yesterday to be the streak anchor (if today not completed)
	start := cur
	if !completed[cur.Format("2006-01-02")] {
		start = cur.AddDate(0, 0, -1)
	}

	for i := 0; i < 365; i++ {
		day := start.AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		if completed[key] {
			streak++
		} else if i == 0 {
			// First day not completed, streak is 0
			return 0
		} else {
			break
		}
	}
	return streak
}

func weeklyStreak(completed map[string]bool, asOf time.Time, target int) int {
	// Count completions per ISO week
	type weekKey struct {
		year int
		week int
	}

	compsPerWeek := make(map[weekKey]int)
	for d := range completed {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		y, w := t.ISOWeek()
		compsPerWeek[weekKey{y, w}]++
	}

	// Current week
	curYear, curWeek := asOf.ISOWeek()

	// Check if current week has enough completions — if not, try previous week
	startYear, startWeek := curYear, curWeek
	if compsPerWeek[weekKey{curYear, curWeek}] < target {
		// Go back one week
		prev := asOf.AddDate(0, 0, -7)
		startYear, startWeek = prev.ISOWeek()
	}

	streak := 0
	for i := 0; i < 52; i++ {
		wk := weekKey{startYear, startWeek}
		if compsPerWeek[wk] >= target {
			streak++
		} else if i == 0 {
			return 0
		} else {
			break
		}
		// Move back one week
		prev := time.Date(startYear, 1, 1, 0, 0, 0, 0, time.UTC)
		_, startWeek = prev.ISOWeek()
	}

	return streak
}

func (s *HabitService) ToggleCompletion(ctx context.Context, habitID, userEmail, dateStr string) (bool, error) {
	_, err := s.repo.FindByID(ctx, habitID, userEmail)
	if err != nil {
		return false, fmt.Errorf("habit not found")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		date = time.Now().UTC()
	}

	existing, err := s.repo.GetCompletion(ctx, habitID, date)
	if err == nil && existing != nil {
		// Remove (uncomplete)
		if err := s.repo.DeleteCompletion(ctx, habitID, date); err != nil {
			return false, fmt.Errorf("delete completion: %w", err)
		}
		return false, nil
	}

	comp := &domain.HabitCompletion{
		ID:            uuid.New().String(),
		HabitID:       habitID,
		CompletedDate: date,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.repo.SaveCompletion(ctx, comp); err != nil {
		return false, fmt.Errorf("save completion: %w", err)
	}

	return true, nil
}

func (s *HabitService) Archive(ctx context.Context, id, userEmail string) error {
	if err := s.repo.Archive(ctx, id, userEmail); err != nil {
		return fmt.Errorf("archive habit: %w", err)
	}
	return nil
}

func (s *HabitService) GetCompletions(ctx context.Context, habitID, userEmail string, from, to time.Time) ([]*domain.HabitCompletion, error) {
	// Verify ownership
	if _, err := s.repo.FindByID(ctx, habitID, userEmail); err != nil {
		return nil, fmt.Errorf("habit not found")
	}

	comps, err := s.repo.GetCompletionsInRange(ctx, habitID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get completions: %w", err)
	}
	return comps, nil
}

type CompletionsGrouped struct {
	Completions map[string][]string `json:"completions"`
	TotalHabits int                 `json:"totalHabits"`
}

func (s *HabitService) GetAllCompletionsGrouped(ctx context.Context, userEmail string, from, to time.Time) (*CompletionsGrouped, error) {
	compsByHabit, err := s.repo.GetAllCompletionsInRange(ctx, userEmail, from, to)
	if err != nil {
		return nil, fmt.Errorf("get all completions: %w", err)
	}

	totalHabits := len(compsByHabit)
	grouped := make(map[string][]string)

	for habitID, comps := range compsByHabit {
		for _, c := range comps {
			dateKey := c.CompletedDate.Format("2006-01-02")
			grouped[dateKey] = append(grouped[dateKey], habitID)
		}
	}

	return &CompletionsGrouped{
		Completions: grouped,
		TotalHabits: totalHabits,
	}, nil
}
