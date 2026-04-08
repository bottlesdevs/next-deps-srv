package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestUserCRUD(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	// Create
	u, err := s.CreateUser(ctx, models.User{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Roles:        []string{"contributor"},
		Enabled:      true,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	// Get
	got, err := s.GetUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("got Username %q, want alice", got.Username)
	}

	// GetByEmail
	byEmail, err := s.GetUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if byEmail.ID != u.ID {
		t.Errorf("email lookup returned wrong user")
	}

	// GetByUsername
	byUsername, err := s.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if byUsername.ID != u.ID {
		t.Errorf("username lookup returned wrong user")
	}

	// Update
	u.Bio = "Hello world"
	if err := s.UpdateUser(ctx, u); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	updated, _ := s.GetUser(ctx, u.ID)
	if updated.Bio != "Hello world" {
		t.Errorf("expected updated bio, got %q", updated.Bio)
	}

	// List
	users, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	// Delete
	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	users, _ = s.ListUsers(ctx)
	if len(users) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(users))
	}
}

func TestDepCRUD(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	dep, err := s.CreateDep(ctx, models.Dependency{
		Name:        "openssl",
		Status:      "pending_review",
		SubmittedBy: "user-1",
		Manifest:    models.Manifest{Name: "openssl", URL: "http://example.com", ExpectedHash: "abc"},
	})
	if err != nil {
		t.Fatalf("CreateDep: %v", err)
	}
	if dep.ID == "" {
		t.Fatal("expected ID")
	}

	got, err := s.GetDep(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetDep: %v", err)
	}
	if got.Name != "openssl" {
		t.Errorf("expected openssl, got %q", got.Name)
	}

	dep.Status = "approved"
	if err := s.UpdateDep(ctx, dep); err != nil {
		t.Fatalf("UpdateDep: %v", err)
	}

	deps, _ := s.ListAllDeps(ctx)
	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}
}

func TestCounters(t *testing.T) {
	ctx := context.Background()
	s := openStore(t)

	if s.CountUsers(ctx) != 0 {
		t.Error("expected 0 users initially")
	}

	s.CreateUser(ctx, models.User{Username: "a", Email: "a@a.com", Roles: []string{"contributor"}, Enabled: true})
	s.CreateUser(ctx, models.User{Username: "b", Email: "b@b.com", Roles: []string{"viewer"}, Enabled: true})

	if s.CountUsers(ctx) != 2 {
		t.Errorf("expected 2 users, got %d", s.CountUsers(ctx))
	}
}
