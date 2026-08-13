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
	"github.com/google/uuid"
)

const testDigest = "b50dc50ec7f41d58b115a6b685d4d1315ba3c797bd3aa0f49213f2703cb82388"

// submitBody mirrors the submit payload: a catalog entry plus its kind and
// metadata. The server assigns the entry id, so it is not sent.
type submitBody struct {
	models.CatalogEntry
	Kind        models.Kind `json:"kind"`
	Category    string      `json:"category"`
	Description string      `json:"description"`
	License     string      `json:"license"`
}

// testEntry builds a minimal dependency submission.
func testEntry(name string) submitBody {
	return submitBody{
		Kind: models.KindDependency,
		CatalogEntry: models.CatalogEntry{
			Name:    name,
			Version: "1.0.0",
			Artifacts: []models.CatalogArtifact{{
				URL:      "https://example.com/" + name + ".zip",
				FileName: name + ".zip",
				Checksum: models.Checksum{Algorithm: "sha256", Value: testDigest},
				Platform: &models.Target{OS: models.OSWindows, Arch: models.ArchX86_64},
			}},
		},
	}
}

// testComponent is the same release published as a component, which the
// schema requires to carry a slot.
func testComponent(name string, slot models.Slot) submitBody {
	b := testEntry(name)
	b.Kind = models.KindComponent
	b.Slot = &slot
	return b
}

func submitDepReq(t *testing.T, ts *httptest.Server, token string, body submitBody) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
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
	resp := submitDepReq(t, ts, "", testEntry("test"))
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

	resp := submitDepReq(t, ts, token, testEntry("mylib"))
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
		Kind:        models.KindDependency,
		Entry:       testEntry("testdep").CatalogEntry,
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

	body := testEntry("badlib")
	body.Artifacts[0].Checksum = models.Checksum{Algorithm: "md5", Value: "abc"}

	resp := submitDepReq(t, ts, token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for unsupported checksum algorithm, got %d", resp.StatusCode)
	}
}

func TestCatalog_SeparatesComponentsFromDependencies(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()

	slot := models.SlotRunner
	comp := testComponent("wine", slot).CatalogEntry
	comp.ID = uuid.NewString()
	if _, err := s.CreateDep(ctx, models.Dependency{
		ID: comp.ID, Name: "wine", Kind: models.KindComponent,
		Status: "built", SubmittedBy: "user-1", Entry: comp,
	}); err != nil {
		t.Fatal(err)
	}

	depEntry := testEntry("zlib").CatalogEntry
	depEntry.ID = uuid.NewString()
	if _, err := s.CreateDep(ctx, models.Dependency{
		ID: depEntry.ID, Name: "zlib", Kind: models.KindDependency,
		Status: "built", SubmittedBy: "user-1", Entry: depEntry,
	}); err != nil {
		t.Fatal(err)
	}

	// Pending entries must not reach either published catalog.
	pending := testEntry("secret").CatalogEntry
	pending.ID = uuid.NewString()
	if _, err := s.CreateDep(ctx, models.Dependency{
		ID: pending.ID, Name: "secret", Kind: models.KindDependency,
		Status: "pending_review", SubmittedBy: "user-1", Entry: pending,
	}); err != nil {
		t.Fatal(err)
	}

	fetch := func(path string, kind models.Kind) models.Catalog {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
		}
		var doc models.Catalog
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		if doc.SchemaVersion != models.SchemaVersion {
			t.Errorf("%s: expected schema_version %d, got %d", path, models.SchemaVersion, doc.SchemaVersion)
		}
		for _, e := range doc.Entries {
			if err := e.Validate(kind); err != nil {
				t.Errorf("%s: published entry is not schema-valid: %v", path, err)
			}
		}
		return doc
	}

	comps := fetch("/api/v1/catalog/components", models.KindComponent)
	if len(comps.Entries) != 1 || comps.Entries[0].Name != "wine" {
		t.Fatalf("expected only the component, got %+v", comps.Entries)
	}
	if comps.Entries[0].Slot == nil || *comps.Entries[0].Slot != models.SlotRunner {
		t.Error("component entry lost its slot")
	}

	deps := fetch("/api/v1/catalog/dependencies", models.KindDependency)
	if len(deps.Entries) != 1 || deps.Entries[0].Name != "zlib" {
		t.Fatalf("expected only the built dependency, got %+v", deps.Entries)
	}
	if deps.Entries[0].Slot != nil {
		t.Error("dependency entry must not carry a slot")
	}
}

// The server owns the entry id, so a client-supplied one is ignored.
func TestSubmitDep_ServerAssignsEntryID(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()
	hash, _ := auth.HashPassword("pw")
	user, _ := s.CreateUser(ctx, models.User{
		Username: "contributor3", Email: "c3@example.com", PasswordHash: hash,
		Roles: []string{"contributor"}, Enabled: true, CreatedAt: time.Now(),
	})
	token, _ := auth.IssueToken(user, testSecret)

	body := testEntry("spoof")
	body.ID = "not-a-uuid-at-all"

	resp := submitDepReq(t, ts, token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var dep models.Dependency
	json.NewDecoder(resp.Body).Decode(&dep)
	if dep.Entry.ID != dep.ID {
		t.Errorf("entry id %q should match record id %q", dep.Entry.ID, dep.ID)
	}
	if _, err := uuid.Parse(dep.Entry.ID); err != nil {
		t.Errorf("entry id is not a uuid: %q", dep.Entry.ID)
	}
}
