package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/api"
	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/middleware"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/queue"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
)

const testSecret = "supersecret"

func setup(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir, err := os.MkdirTemp("", "ndeps-api-test-*")
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	local := models.LocalStorageConfig{
		BucketRoot: dir + "/bucket",
		DedupRoot:  dir + "/dedup",
	}
	backend, err := bucket.NewLocalBackend(local)
	if err != nil {
		t.Fatal(err)
	}
	bq := queue.New(s, backend, nil)
	srv := api.NewServer(s, bq, backend, nil, testSecret, dir)
	rl := middleware.NewRateLimiter(models.RateLimitConfig{Enabled: false})
	ts := httptest.NewServer(srv.Handler(rl))
	t.Cleanup(func() {
		ts.Close()
		s.Close()
		os.RemoveAll(dir)
	})
	return ts, s
}

func createAdminToken(t *testing.T, s *store.Store) string {
	t.Helper()
	hash, _ := auth.HashPassword("password")
	u, err := s.CreateUser(context.Background(), models.User{
		Username:     "admin",
		Email:        "admin@test.com",
		PasswordHash: hash,
		Roles:        []string{"admin"},
		Enabled:      true,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := auth.IssueToken(u, testSecret)
	return tok
}

func do(t *testing.T, ts *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, ts.URL+path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func TestLogin(t *testing.T) {
	ts, s := setup(t)
	_ = createAdminToken(t, s)

	resp := do(t, ts, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "password",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: got %d", resp.StatusCode)
	}
	var out map[string]string
	json.NewDecoder(resp.Body).Decode(&out)
	if out["token"] == "" {
		t.Error("expected token in response")
	}
}

func TestLoginBadCredentials(t *testing.T) {
	ts, s := setup(t)
	_ = createAdminToken(t, s)

	resp := do(t, ts, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetMe(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "GET", "/api/v1/auth/me", nil, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("getMe: got %d", resp.StatusCode)
	}
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	if out["username"] != "admin" {
		t.Errorf("expected username admin, got %v", out["username"])
	}
}

func TestGetMeUnauthorized(t *testing.T) {
	ts, _ := setup(t)
	resp := do(t, ts, "GET", "/api/v1/auth/me", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestChangePassword(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "PUT", "/api/v1/auth/me/password", map[string]string{
		"current_password": "password",
		"new_password":     "new-password",
	}, tok)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("changePassword: got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = do(t, ts, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "password",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("old password: expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = do(t, ts, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "new-password",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("new password: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestChangePasswordWrongCurrentPassword(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "PUT", "/api/v1/auth/me/password", map[string]string{
		"current_password": "wrong-password",
		"new_password":     "new-password",
	}, tok)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = do(t, ts, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "password",
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("original password: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestChangePasswordRejectsInvalidRequest(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "PUT", "/api/v1/auth/me/password", map[string]string{
		"current_password": "password",
		"new_password":     "short",
	}, tok)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("short password: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = do(t, ts, "PUT", "/api/v1/auth/me/password", map[string]string{
		"current_password": "password",
		"new_password":     "new-password",
		"password":         "unexpected",
	}, tok)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown field: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestChangePasswordUnauthorized(t *testing.T) {
	ts, _ := setup(t)
	resp := do(t, ts, "PUT", "/api/v1/auth/me/password", map[string]string{
		"current_password": "password",
		"new_password":     "new-password",
	}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestListDepsEmpty(t *testing.T) {
	ts, _ := setup(t)
	resp := do(t, ts, "GET", "/api/v1/deps", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("listDeps: got %d", resp.StatusCode)
	}
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	if out["total"].(float64) != 0 {
		t.Errorf("expected 0 deps, got %v", out["total"])
	}
}

func TestAdminStats(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "GET", "/api/v1/admin/stats", nil, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("adminStats: got %d", resp.StatusCode)
	}
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	if _, ok := out["users"]; !ok {
		t.Error("expected 'users' field in stats")
	}
}

func TestAdminStatsUnauthorized(t *testing.T) {
	ts, _ := setup(t)
	resp := do(t, ts, "GET", "/api/v1/admin/stats", nil, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestSubmitDepForbiddenForViewer(t *testing.T) {
	ts, s := setup(t)

	hash, _ := auth.HashPassword("pw")
	viewer, _ := s.CreateUser(context.Background(), models.User{
		Username: "viewer", Email: "v@test.com", PasswordHash: hash,
		Roles: []string{"viewer"}, Enabled: true, CreatedAt: time.Now(),
	})
	tok, _ := auth.IssueToken(viewer, testSecret)

	resp := do(t, ts, "POST", "/api/v1/deps", testEntry("test"), tok)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestCommunityPostCRUD(t *testing.T) {
	ts, s := setup(t)
	tok := createAdminToken(t, s)

	resp := do(t, ts, "POST", "/api/v1/community", map[string]string{"body": "hello world"}, tok)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createPost: got %d", resp.StatusCode)
	}
	var post map[string]any
	json.NewDecoder(resp.Body).Decode(&post)
	id := post["id"].(string)

	resp = do(t, ts, "GET", "/api/v1/community", nil, tok)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("listPosts: got %d", resp.StatusCode)
	}
	var listOut map[string]any
	json.NewDecoder(resp.Body).Decode(&listOut)
	if listOut["total"].(float64) != 1 {
		t.Errorf("expected 1 post, got %v", listOut["total"])
	}

	resp = do(t, ts, "POST", "/api/v1/community/"+id+"/replies", map[string]string{"body": "reply"}, tok)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createReply: got %d", resp.StatusCode)
	}

	resp = do(t, ts, "DELETE", "/api/v1/community/"+id, nil, tok)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("deletePost: got %d", resp.StatusCode)
	}
}
