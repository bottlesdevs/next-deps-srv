package store

import (
	"context"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateDep(ctx context.Context, d models.Dependency) (models.Dependency, error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	return d, s.Deps.Put(ctx, d.ID, d, 0)
}

func (s *Store) GetDep(ctx context.Context, id string) (models.Dependency, error) {
	return s.Deps.Get(ctx, id)
}

func (s *Store) UpdateDep(ctx context.Context, d models.Dependency) error {
	d.UpdatedAt = time.Now()
	return s.Deps.Put(ctx, d.ID, d, 0)
}

func (s *Store) DeleteDep(ctx context.Context, id string) error {
	return s.Deps.Delete(ctx, id)
}

func (s *Store) ListApprovedDeps(ctx context.Context) ([]models.Dependency, error) {
	return s.Deps.GetByIndex(ctx, "status", "built").All()
}

func (s *Store) ListPendingDeps(ctx context.Context) ([]models.Dependency, error) {
	return s.Deps.GetByIndex(ctx, "status", "pending_review").All()
}

func (s *Store) ListAllDeps(ctx context.Context) ([]models.Dependency, error) {
	return s.Deps.GetByIndex(ctx, "all", "all").All()
}

func (s *Store) CountDeps(ctx context.Context) int {
	deps, _ := s.Deps.GetByIndex(ctx, "all", "all").All()
	return len(deps)
}
