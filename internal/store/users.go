package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func (s *Store) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	return u, s.Users.Put(ctx, u.ID, u, 0)
}

func (s *Store) GetUser(ctx context.Context, id string) (models.User, error) {
	return s.Users.Get(ctx, id)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	results, err := s.Users.GetByIndex(ctx, "email", email).All()
	if err != nil || len(results) == 0 {
		return models.User{}, fmt.Errorf("user not found")
	}
	return results[0], nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	results, err := s.Users.GetByIndex(ctx, "username", username).All()
	if err != nil || len(results) == 0 {
		return models.User{}, fmt.Errorf("user not found")
	}
	return results[0], nil
}

func (s *Store) UpdateUser(ctx context.Context, u models.User) error {
	u.UpdatedAt = time.Now()
	return s.Users.Put(ctx, u.ID, u, 0)
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	return s.Users.Delete(ctx, id)
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.Users.GetByIndex(ctx, "all", "all").All()
}

func (s *Store) CountUsers(ctx context.Context) int {
	users, _ := s.Users.GetByIndex(ctx, "all", "all").All()
	return len(users)
}

func (s *Store) AdminAndModUsers(ctx context.Context) ([]models.User, error) {
	all, err := s.Users.GetByIndex(ctx, "all", "all").All()
	if err != nil {
		return nil, err
	}
	var out []models.User
	for _, u := range all {
		for _, r := range u.Roles {
			if r == "admin" || r == "mod" {
				out = append(out, u)
				break
			}
		}
	}
	return out, nil
}
