package auth_test

import (
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
)

func TestHashAndCheckPassword(t *testing.T) {
	plain := "super-secure-password"
	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == plain {
		t.Fatal("hash must differ from plaintext")
	}

	if !auth.CheckPassword(hash, plain) {
		t.Error("CheckPassword should return true for correct password")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	h1, _ := auth.HashPassword("pass")
	h2, _ := auth.HashPassword("pass")
	if h1 == h2 {
		t.Error("bcrypt hashes should differ (different salts)")
	}
}
