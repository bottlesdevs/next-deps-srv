package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateFile(ctx context.Context, f models.BucketFile) (models.BucketFile, error) {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	f.CreatedAt = time.Now()
	f.UpdatedAt = f.CreatedAt
	return f, s.Files.Put(ctx, f.ID, f, 0)
}

func (s *Store) GetFile(ctx context.Context, id string) (models.BucketFile, error) {
	return s.Files.Get(ctx, id)
}

func (s *Store) GetFileByName(ctx context.Context, name string) (models.BucketFile, error) {
	results, err := s.Files.GetByIndex(ctx, "name", name).All()
	if err != nil || len(results) == 0 {
		return models.BucketFile{}, fmt.Errorf("file not found")
	}
	return results[0], nil
}

func (s *Store) UpdateFile(ctx context.Context, f models.BucketFile) error {
	f.UpdatedAt = time.Now()
	return s.Files.Put(ctx, f.ID, f, 0)
}

func (s *Store) CountFiles(ctx context.Context) int {
	files, _ := s.Files.GetByIndex(ctx, "all", "all").All()
	return len(files)
}

func (s *Store) CreateRevision(ctx context.Context, r models.FileRevision) (models.FileRevision, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	r.CreatedAt = time.Now()
	return r, s.Revs.Put(ctx, r.ID, r, 0)
}

func (s *Store) GetRevision(ctx context.Context, id string) (models.FileRevision, error) {
	return s.Revs.Get(ctx, id)
}

func (s *Store) RevisionsByFile(ctx context.Context, fileID string) ([]models.FileRevision, error) {
	return s.Revs.GetByIndex(ctx, "file_id", fileID).
		Sort(func(a, b models.FileRevision) bool { return a.RevisionNum < b.RevisionNum }).All()
}

func (s *Store) CountRevisions(ctx context.Context) int {
	revs, _ := s.Revs.GetByIndex(ctx, "all", "all").All()
	return len(revs)
}
