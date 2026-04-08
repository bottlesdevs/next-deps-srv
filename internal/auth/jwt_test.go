package auth_test

import (
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

func TestIssueAndValidate(t *testing.T) {
	u := models.User{ID: "u1", Username: "alice", Roles: []string{"admin", "mod"}}
	token, err := auth.IssueToken(u, "testsecret")
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	claims, err := auth.ValidateToken("testsecret", token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != "u1" {
		t.Errorf("got UserID %q, want u1", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("got Username %q, want alice", claims.Username)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "admin" {
		t.Errorf("unexpected roles: %v", claims.Roles)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	u := models.User{ID: "u1", Username: "bob", Roles: []string{"contributor"}}
	token, _ := auth.IssueToken(u, "secret1")
	if _, err := auth.ValidateToken("secret2", token); err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestHasRole(t *testing.T) {
	claims := &auth.Claims{Roles: []string{"admin", "mod"}}
	if !auth.HasRole(claims, "admin") {
		t.Error("expected HasRole admin to be true")
	}
	if !auth.HasRole(claims, "mod", "contributor") {
		t.Error("expected HasRole mod|contributor to be true")
	}
	if auth.HasRole(claims, "contributor") {
		t.Error("expected HasRole contributor to be false")
	}
}
