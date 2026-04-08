package store

import (
	"context"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateJob(ctx context.Context, j models.BuildJob) (models.BuildJob, error) {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	j.StartedAt = time.Now()
	return j, s.Jobs.Put(ctx, j.ID, j, 0)
}

func (s *Store) GetJob(ctx context.Context, id string) (models.BuildJob, error) {
	return s.Jobs.Get(ctx, id)
}

func (s *Store) UpdateJob(ctx context.Context, j models.BuildJob) error {
	return s.Jobs.Put(ctx, j.ID, j, 0)
}

func (s *Store) ListJobs(ctx context.Context) ([]models.BuildJob, error) {
	return s.Jobs.GetByIndex(ctx, "all", "all").All()
}

func (s *Store) JobsByDep(ctx context.Context, depID string) ([]models.BuildJob, error) {
	return s.Jobs.GetByIndex(ctx, "dep_id", depID).All()
}
