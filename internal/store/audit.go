package store

import (
	"context"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) Log(ctx context.Context, entry models.AuditEntry) error {
	entry.ID = uuid.NewString()
	entry.CreatedAt = time.Now()
	return s.Audit.Put(ctx, entry.ID, entry, 0)
}

func (s *Store) ListAudit(ctx context.Context) ([]models.AuditEntry, error) {
	return s.Audit.GetByIndex(ctx, "all", "all").
		Sort(func(a, b models.AuditEntry) bool { return a.CreatedAt.After(b.CreatedAt) }).All()
}

func (s *Store) AuditByUser(ctx context.Context, userID string) ([]models.AuditEntry, error) {
	return s.Audit.GetByIndex(ctx, "user_id", userID).
		Sort(func(a, b models.AuditEntry) bool { return a.CreatedAt.After(b.CreatedAt) }).All()
}
