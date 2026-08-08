package application

import (
	"context"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// CategoryService manages the category metadata table. It is unscoped: the app
// is single-user, so there is no per-user filtering.
type CategoryService struct {
	repo domain.CategoryRepository
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// List returns every category, active and inactive, ordered by name.
func (s *CategoryService) List(ctx context.Context) ([]*domain.Category, error) {
	return s.repo.List(ctx)
}

// Update reclassifies an existing category. The classification is validated
// against the taxonomy; essential and active are independent flags.
func (s *CategoryService) Update(ctx context.Context, name string, classification domain.Classification, essential, active bool) (*domain.Category, error) {
	if !domain.ValidClassification(classification) {
		return nil, fmt.Errorf("invalid classification %q", classification)
	}

	// Load the existing row so created_at survives and the update is a true
	// partial edit rather than a blind overwrite.
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("find category: %w", err)
	}

	existing.Classification = classification
	existing.Essential = essential
	existing.Active = active
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update category: %w", err)
	}
	return existing, nil
}
