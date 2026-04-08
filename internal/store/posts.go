package store

import (
	"context"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreatePost(ctx context.Context, p models.CommunityPost) (models.CommunityPost, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	return p, s.Posts.Put(ctx, p.ID, p, 0)
}

func (s *Store) GetPost(ctx context.Context, id string) (models.CommunityPost, error) {
	return s.Posts.Get(ctx, id)
}

func (s *Store) UpdatePost(ctx context.Context, p models.CommunityPost) error {
	p.UpdatedAt = time.Now()
	return s.Posts.Put(ctx, p.ID, p, 0)
}

func (s *Store) ListTopPosts(ctx context.Context) ([]models.CommunityPost, error) {
	return s.Posts.GetByIndex(ctx, "parent_id", "").
		Filter(func(p models.CommunityPost) bool { return !p.Deleted }).
		Sort(func(a, b models.CommunityPost) bool { return a.CreatedAt.After(b.CreatedAt) }).All()
}

func (s *Store) ListReplies(ctx context.Context, parentID string) ([]models.CommunityPost, error) {
	return s.Posts.GetByIndex(ctx, "parent_id", parentID).
		Filter(func(p models.CommunityPost) bool { return !p.Deleted }).
		Sort(func(a, b models.CommunityPost) bool { return a.CreatedAt.Before(b.CreatedAt) }).All()
}

func (s *Store) CountReplies(ctx context.Context, parentID string) (int, error) {
	replies, err := s.Posts.GetByIndex(ctx, "parent_id", parentID).
		Filter(func(p models.CommunityPost) bool { return !p.Deleted }).All()
	return len(replies), err
}
