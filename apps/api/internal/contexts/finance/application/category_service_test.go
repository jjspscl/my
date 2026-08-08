package application

import (
	"context"
	"errors"
	"testing"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type mockCategoryRepo struct {
	categories []*domain.Category
	err        error
}

func (m *mockCategoryRepo) List(_ context.Context) ([]*domain.Category, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.categories, nil
}

func (m *mockCategoryRepo) FindByName(_ context.Context, name string) (*domain.Category, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, c := range m.categories {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, errors.New("category not found")
}

func (m *mockCategoryRepo) Update(_ context.Context, c *domain.Category) error {
	if m.err != nil {
		return m.err
	}
	for i, existing := range m.categories {
		if existing.Name == c.Name {
			m.categories[i] = c
			return nil
		}
	}
	return errors.New("category not found")
}

func categoryFixture() *mockCategoryRepo {
	return &mockCategoryRepo{categories: []*domain.Category{
		{Name: "Food", Classification: domain.ClassificationUnclassified, Essential: false, Active: true},
		{Name: "Bills", Classification: domain.ClassificationUnclassified, Essential: false, Active: true},
	}}
}

func TestCategoryServiceList(t *testing.T) {
	repo := categoryFixture()
	svc := NewCategoryService(repo)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(got))
	}
}

func TestCategoryServiceUpdateClassifies(t *testing.T) {
	repo := categoryFixture()
	svc := NewCategoryService(repo)

	updated, err := svc.Update(context.Background(), "Food", domain.ClassificationNeeds, true, true)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Classification != domain.ClassificationNeeds {
		t.Errorf("classification = %q, want needs", updated.Classification)
	}
	if !updated.Essential {
		t.Error("expected essential = true")
	}
	if !updated.Active {
		t.Error("expected active = true")
	}
}

func TestCategoryServiceUpdateRejectsInvalidClassification(t *testing.T) {
	svc := NewCategoryService(categoryFixture())
	if _, err := svc.Update(context.Background(), "Food", "bogus", false, true); err == nil {
		t.Fatal("expected error for invalid classification")
	}
}

func TestCategoryServiceUpdateUnknownCategory(t *testing.T) {
	svc := NewCategoryService(categoryFixture())
	if _, err := svc.Update(context.Background(), "Nope", domain.ClassificationNeeds, false, true); err == nil {
		t.Fatal("expected error for unknown category")
	}
}

func TestCategoryServiceUpdatePreservesCreatedAt(t *testing.T) {
	repo := categoryFixture()
	svc := NewCategoryService(repo)

	before := repo.categories[0].CreatedAt
	updated, err := svc.Update(context.Background(), "Food", domain.ClassificationWants, false, false)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.CreatedAt.Equal(before) {
		t.Error("Update must not clobber created_at")
	}
	if updated.Active {
		t.Error("expected active = false after update")
	}
}