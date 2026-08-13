package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/auth"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

// testItem builds a minimal catalog item that satisfies Item.Validate.
func testItem(name string) models.Item {
	return models.Item{
		ID:      name,
		Name:    name,
		Version: "1.0.0",
		Kind:    &models.Kind{Type: "library", Flavour: "shared"},
		Artifacts: []models.Artifact{{
			URL:           "https://example.com/" + name + ".zip",
			FileName:      name + ".zip",
			Checksum:      &models.Checksum{Algorithm: "sha256", Value: "deadbeef"},
			Size:          1024,
			Platform:      &models.Platform{OS: "windows", Arch: "x86_64"},
			ComponentRoot: name,
		}},
	}
}

func submitDepReq(t *testing.T, ts *httptest.Server, token string, item models.Item) *http.Response {
	t.Helper()
	b, _ := json.Marshal(item)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/deps", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/v1/deps: %v", err)
	}
	return resp
}

func TestSubmitDep_RequiresAuth(t *testing.T) {
	ts, _ := setup(t)
	resp := submitDepReq(t, ts, "", testItem("test"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}
}

func TestSubmitDep_ContributorCanSubmit(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()
	hash, _ := auth.HashPassword("pw")
	user, _ := s.CreateUser(ctx, models.User{
		Username:     "contributor1",
		Email:        "c1@example.com",
		PasswordHash: hash,
		Roles:        []string{"contributor"},
		Enabled:      true,
		CreatedAt:    time.Now(),
	})
	token, _ := auth.IssueToken(user, testSecret)

	resp := submitDepReq(t, ts, token, testItem("mylib"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	var dep models.Dependency
	json.NewDecoder(resp.Body).Decode(&dep)
	if dep.ID == "" {
		t.Error("expected dep ID in response")
	}
	if dep.Status != "pending_review" {
		t.Errorf("expected pending_review, got %s", dep.Status)
	}
}

func TestGetDep_Exists(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()

	dep, _ := s.CreateDep(ctx, models.Dependency{
		Name:        "testdep",
		Status:      "built",
		SubmittedBy: "user-1",
		Item:        testItem("testdep"),
	})

	resp, err := http.Get(ts.URL + "/api/v1/deps/" + dep.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var out models.Dependency
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Name != "testdep" {
		t.Errorf("expected testdep, got %q", out.Name)
	}
}

func TestSubmitDep_RejectsInvalidItem(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()
	hash, _ := auth.HashPassword("pw")
	user, _ := s.CreateUser(ctx, models.User{
		Username:     "contributor2",
		Email:        "c2@example.com",
		PasswordHash: hash,
		Roles:        []string{"contributor"},
		Enabled:      true,
		CreatedAt:    time.Now(),
	})
	token, _ := auth.IssueToken(user, testSecret)

	item := testItem("badlib")
	item.Artifacts[0].Checksum = &models.Checksum{Algorithm: "crc32", Value: "abc"}

	resp := submitDepReq(t, ts, token, item)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported checksum algorithm, got %d", resp.StatusCode)
	}
}

func TestCatalog_EmitsSchemaDocument(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()

	built := testItem("zlib")
	if _, err := s.CreateDep(ctx, models.Dependency{
		Name: "zlib", Status: "built", SubmittedBy: "user-1", Item: built,
	}); err != nil {
		t.Fatal(err)
	}
	// Pending deps must not reach the published catalog.
	if _, err := s.CreateDep(ctx, models.Dependency{
		Name: "secret", Status: "pending_review", SubmittedBy: "user-1", Item: testItem("secret"),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(ts.URL + "/api/v1/catalog")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var doc models.Catalog
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != models.SchemaVersion {
		t.Errorf("expected schema_version %d, got %d", models.SchemaVersion, doc.SchemaVersion)
	}
	if len(doc.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(doc.Items))
	}
	if doc.Items[0].ID != "zlib" {
		t.Errorf("expected zlib, got %q", doc.Items[0].ID)
	}
	if err := doc.Items[0].Validate(); err != nil {
		t.Errorf("published item is not schema-valid: %v", err)
	}
}
