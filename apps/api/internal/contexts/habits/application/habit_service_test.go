package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/habits/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- mocks ----

type mockHabitRepo struct {
	habits                  map[string]*domain.Habit
	completions             []*domain.HabitCompletion
	saveHabitFn             func(ctx context.Context, h *domain.Habit) error
	listActiveFn            func(ctx context.Context, userEmail string) ([]*domain.Habit, error)
	findByIDFn              func(ctx context.Context, id, userEmail string) (*domain.Habit, error)
	archiveFn               func(ctx context.Context, id, userEmail string) error
	getCompletionsInRangeFn func(ctx context.Context, habitID string, from, to time.Time) ([]*domain.HabitCompletion, error)
}

func newMockHabitRepo() *mockHabitRepo {
	return &mockHabitRepo{
		habits:      make(map[string]*domain.Habit),
		completions: make([]*domain.HabitCompletion, 0),
	}
}

func (m *mockHabitRepo) SaveHabit(ctx context.Context, h *domain.Habit) error {
	if m.saveHabitFn != nil {
		return m.saveHabitFn(ctx, h)
	}
	h.Archived = false
	m.habits[h.ID] = h
	return nil
}

func (m *mockHabitRepo) ListActive(ctx context.Context, userEmail string) ([]*domain.Habit, error) {
	if m.listActiveFn != nil {
		return m.listActiveFn(ctx, userEmail)
	}
	var result []*domain.Habit
	for _, h := range m.habits {
		if h.UserEmail == userEmail && !h.Archived {
			result = append(result, h)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) FindByID(ctx context.Context, id, userEmail string) (*domain.Habit, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id, userEmail)
	}
	h, ok := m.habits[id]
	if !ok || h.UserEmail != userEmail {
		return nil, errors.New("habit not found")
	}
	return h, nil
}

func (m *mockHabitRepo) Archive(ctx context.Context, id, userEmail string) error {
	if m.archiveFn != nil {
		return m.archiveFn(ctx, id, userEmail)
	}
	h, ok := m.habits[id]
	if !ok || h.UserEmail != userEmail {
		return errors.New("habit not found")
	}
	h.Archived = true
	return nil
}

func (m *mockHabitRepo) SaveCompletion(ctx context.Context, c *domain.HabitCompletion) error {
	m.completions = append(m.completions, c)
	return nil
}

func (m *mockHabitRepo) DeleteCompletion(ctx context.Context, habitID string, date time.Time) error {
	for i, c := range m.completions {
		if c.HabitID == habitID && c.CompletedDate.Format("2006-01-02") == date.Format("2006-01-02") {
			m.completions = append(m.completions[:i], m.completions[i+1:]...)
			return nil
		}
	}
	return errors.New("completion not found")
}

func (m *mockHabitRepo) GetCompletion(ctx context.Context, habitID string, date time.Time) (*domain.HabitCompletion, error) {
	for _, c := range m.completions {
		if c.HabitID == habitID && c.CompletedDate.Format("2006-01-02") == date.Format("2006-01-02") {
			return c, nil
		}
	}
	return nil, errors.New("completion not found")
}

func (m *mockHabitRepo) GetCompletionsInRange(ctx context.Context, habitID string, from, to time.Time) ([]*domain.HabitCompletion, error) {
	if m.getCompletionsInRangeFn != nil {
		return m.getCompletionsInRangeFn(ctx, habitID, from, to)
	}
	var result []*domain.HabitCompletion
	for _, c := range m.completions {
		if c.HabitID == habitID && !c.CompletedDate.Before(from) && !c.CompletedDate.After(to) {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockHabitRepo) GetAllCompletionsInRange(ctx context.Context, userEmail string, from, to time.Time) (map[string][]*domain.HabitCompletion, error) {
	result := make(map[string][]*domain.HabitCompletion)
	for _, c := range m.completions {
		// Find the habit to check user_email
		h, ok := m.habits[c.HabitID]
		if !ok || h.UserEmail != userEmail {
			continue
		}
		if !c.CompletedDate.Before(from) && !c.CompletedDate.After(to) {
			result[c.HabitID] = append(result[c.HabitID], c)
		}
	}
	return result, nil
}

func newTestHabitService() *HabitService {
	return NewHabitService(newMockHabitRepo())
}

// ---- tests ----

func TestCreate_ValidHabit_SavesAndReturns(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:          "Exercise",
		Color:         "green",
		Frequency:     "daily",
		TargetPerWeek: 7,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, h.ID)
	assert.Equal(t, "Exercise", h.Name)
	assert.Equal(t, "green", h.Color)
	assert.Equal(t, domain.FrequencyDaily, h.Frequency)
	assert.Equal(t, 7, h.TargetPerWeek)
	assert.False(t, h.Archived)
}

func TestCreate_EmptyName_ReturnsError(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "",
		Color:     "blue",
		Frequency: "daily",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestCreate_EmptyColor_DefaultsToBlue(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Test",
		Color:     "",
		Frequency: "daily",
	})
	require.NoError(t, err)
	assert.Equal(t, "blue", h.Color)
}

func TestCreate_InvalidFrequency_DefaultsToDaily(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Test",
		Color:     "green",
		Frequency: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, domain.FrequencyDaily, h.Frequency)
}

func TestCreate_WeeklyFrequency_Preserved(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:          "Learn Go",
		Color:         "indigo",
		Frequency:     "weekly",
		TargetPerWeek: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.FrequencyWeekly, h.Frequency)
	assert.Equal(t, 3, h.TargetPerWeek)
}

func TestCreate_ZeroTargetPerWeek_DefaultsToOne(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:          "Test",
		Color:         "blue",
		Frequency:     "daily",
		TargetPerWeek: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, h.TargetPerWeek)
}

func TestToggleCompletion_NotCompleted_CreatesAndReturnsTrue(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	todayStr := time.Now().UTC().Format("2006-01-02")

	// Create a habit first
	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})
	require.NoError(t, err)

	completed, err := svc.ToggleCompletion(ctx, h.ID, "user@test.com", todayStr)
	assert.NoError(t, err)
	assert.True(t, completed)
	assert.Len(t, repo.completions, 1)
}

func TestToggleCompletion_AlreadyCompleted_DeletesAndReturnsFalse(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	todayStr := time.Now().UTC().Format("2006-01-02")

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})
	require.NoError(t, err)

	// Toggle ON
	completed1, err := svc.ToggleCompletion(ctx, h.ID, "user@test.com", todayStr)
	assert.NoError(t, err)
	assert.True(t, completed1)
	assert.Len(t, repo.completions, 1)

	// Toggle OFF (same day)
	completed2, err := svc.ToggleCompletion(ctx, h.ID, "user@test.com", todayStr)
	assert.NoError(t, err)
	assert.False(t, completed2)
	assert.Len(t, repo.completions, 0)
}

func TestToggleCompletion_NonexistentHabit_ReturnsError(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	_, err := svc.ToggleCompletion(ctx, "nonexistent", "user@test.com", "")
	assert.Error(t, err)
}

func TestArchive_ExistingHabit_SetsArchived(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})
	require.NoError(t, err)

	err = svc.Archive(ctx, h.ID, "user@test.com")
	assert.NoError(t, err)

	// Verify it's archived
	habit, err := repo.FindByID(ctx, h.ID, "user@test.com")
	assert.NoError(t, err)
	assert.True(t, habit.Archived)
}

func TestArchive_NonexistentHabit_ReturnsError(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	err := svc.Archive(ctx, "nonexistent", "user@test.com")
	assert.Error(t, err)
}

func TestListWithStatus_ReturnsHabitsWithTodayFlag(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	now := time.Now().UTC()
	todayStr := now.Format("2006-01-02")

	h, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})
	require.NoError(t, err)

	// Toggle ON with explicit today date string
	svc.ToggleCompletion(ctx, h.ID, "user@test.com", todayStr)

	results, err := svc.ListWithStatus(ctx, "user@test.com", now)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].CompletedToday)
}

func TestListWithStatus_WithoutCompletion_ShowsNotCompleted(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})
	require.NoError(t, err)

	results, err := svc.ListWithStatus(ctx, "user@test.com", now)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].CompletedToday)
	assert.Equal(t, 0, results[0].CurrentStreak)
}

func TestGetCompletions_ExistingHabit_ReturnsCompletions(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	now := time.Now()

	h, _ := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})

	svc.ToggleCompletion(ctx, h.ID, "user@test.com", "")

	from := now.AddDate(0, 0, -7)
	to := now.AddDate(0, 0, 1)
	comps, err := svc.GetCompletions(ctx, h.ID, "user@test.com", from, to)
	require.NoError(t, err)
	assert.Len(t, comps, 1)
}

func TestGetCompletions_NonexistentHabit_ReturnsError(t *testing.T) {
	svc := newTestHabitService()
	ctx := context.Background()

	_, err := svc.GetCompletions(ctx, "nonexistent", "user@test.com", time.Now(), time.Now())
	assert.Error(t, err)
}

func TestGetAllCompletionsGrouped_ReturnsMap(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()

	h1, _ := svc.Create(ctx, "user@test.com", CreateHabitInput{Name: "Exercise", Color: "green", Frequency: "daily"})
	h2, _ := svc.Create(ctx, "user@test.com", CreateHabitInput{Name: "Read", Color: "blue", Frequency: "daily"})

	svc.ToggleCompletion(ctx, h1.ID, "user@test.com", "")
	svc.ToggleCompletion(ctx, h2.ID, "user@test.com", "")

	from := time.Now().AddDate(0, 0, -7)
	to := time.Now().AddDate(0, 0, 1)

	result, err := svc.GetAllCompletionsGrouped(ctx, "user@test.com", from, to)
	require.NoError(t, err)

	todayStr := time.Now().Format("2006-01-02")
	assert.Contains(t, result.Completions, todayStr)
	assert.Len(t, result.Completions[todayStr], 2) // both habits completed today
	assert.Equal(t, 2, result.TotalHabits)
}

func TestGetAllCompletionsGrouped_NoCompletions_ReturnsEmpty(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	now := time.Now().UTC()

	svc.Create(ctx, "user@test.com", CreateHabitInput{Name: "Exercise", Color: "green", Frequency: "daily"})

	from := now.AddDate(0, 0, -7)
	to := now.AddDate(0, 0, -1)

	result, err := svc.GetAllCompletionsGrouped(ctx, "user@test.com", from, to)
	require.NoError(t, err)
	assert.Empty(t, result.Completions)
	// TotalHabits is len of compsByHabit map keys — no completions means 0 habits
	assert.Equal(t, 0, result.TotalHabits)
}

func TestListWithStatus_StreakCalculation(t *testing.T) {
	repo := newMockHabitRepo()
	svc := NewHabitService(repo)
	ctx := context.Background()
	now := time.Now().UTC()
	todayStr := now.Format("2006-01-02")
	yesterdayStr := now.AddDate(0, 0, -1).Format("2006-01-02")

	h, _ := svc.Create(ctx, "user@test.com", CreateHabitInput{
		Name:      "Exercise",
		Color:     "green",
		Frequency: "daily",
	})

	// Complete today and yesterday with explicit date strings
	svc.ToggleCompletion(ctx, h.ID, "user@test.com", todayStr)
	svc.ToggleCompletion(ctx, h.ID, "user@test.com", yesterdayStr)

	results, err := svc.ListWithStatus(ctx, "user@test.com", now)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 2, results[0].CurrentStreak)
}
