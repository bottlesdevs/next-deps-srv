package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
	"github.com/google/uuid"
)

func putBuiltEntry(t *testing.T, s *store.Store, kind models.Kind, name, version string, slot *models.Slot, requirements []models.Requirement, artifacts []models.CatalogArtifact, updatedAt time.Time) models.Dependency {
	t.Helper()
	id := uuid.NewString()
	dep := models.Dependency{
		ID: id, Name: name, Kind: kind, Status: "built", SubmittedBy: "user-1",
		Entry: models.CatalogEntry{
			ID: id, Name: name, Version: version, Slot: slot,
			Requirements: requirements, Artifacts: artifacts,
		},
		CreatedAt: updatedAt, UpdatedAt: updatedAt,
	}
	if err := s.Deps.Put(context.Background(), dep.ID, dep, 0); err != nil {
		t.Fatal(err)
	}
	return dep
}

func resolveArtifact(fileName string, target *models.Target) models.CatalogArtifact {
	return models.CatalogArtifact{
		URL:      "https://example.com/" + fileName,
		FileName: fileName,
		Checksum: models.Checksum{Algorithm: "sha256", Value: testDigest},
		Platform: target,
	}
}

func TestResolve_OrdersRequirementsAndFiltersPlatform(t *testing.T) {
	ts, s := setup(t)
	windows := &models.Target{OS: models.OSWindows, Arch: models.ArchX86_64}
	linux := &models.Target{OS: models.OSLinux, Arch: models.ArchX86_64}
	dep := putBuiltEntry(t, s, models.KindDependency, "vcredist", "14", nil, nil,
		[]models.CatalogArtifact{resolveArtifact("vcredist.exe", windows)}, time.Unix(1, 0))
	slot := models.SlotDXVK
	putBuiltEntry(t, s, models.KindComponent, "dxvk", "2.0", &slot,
		[]models.Requirement{{ID: dep.ID}},
		[]models.CatalogArtifact{resolveArtifact("dxvk-win.tar.xz", windows), resolveArtifact("dxvk-linux.tar.xz", linux)}, time.Unix(2, 0))

	resp := do(t, ts, http.MethodPost, "/api/v1/resolve", map[string]any{
		"components": []map[string]string{{"slot": "dxvk", "version": "2.0"}},
		"platform":   map[string]string{"os": "windows", "arch": "x86_64"},
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		SchemaVersion int `json:"schema_version"`
		Entries       []struct {
			Kind      models.Kind              `json:"kind"`
			Name      string                   `json:"name"`
			Artifacts []models.CatalogArtifact `json:"artifacts"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.SchemaVersion != models.SchemaVersion {
		t.Errorf("expected schema version %d, got %d", models.SchemaVersion, out.SchemaVersion)
	}
	if len(out.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out.Entries))
	}
	if out.Entries[0].Name != "vcredist" || out.Entries[1].Name != "dxvk" {
		t.Errorf("requirements must precede roots, got %s then %s", out.Entries[0].Name, out.Entries[1].Name)
	}
	if len(out.Entries[1].Artifacts) != 1 || out.Entries[1].Artifacts[0].FileName != "dxvk-win.tar.xz" {
		t.Errorf("expected only the Windows artifact, got %+v", out.Entries[1].Artifacts)
	}
}

func TestResolve_UsesNewestVersionUnlessVersionIsRequested(t *testing.T) {
	ts, s := setup(t)
	putBuiltEntry(t, s, models.KindDependency, "cabextract", "1.0", nil, nil,
		[]models.CatalogArtifact{resolveArtifact("cabextract-1.zip", nil)}, time.Unix(1, 0))
	putBuiltEntry(t, s, models.KindDependency, "cabextract", "2.0", nil, nil,
		[]models.CatalogArtifact{resolveArtifact("cabextract-2.zip", nil)}, time.Unix(2, 0))

	request := func(selector map[string]string) string {
		t.Helper()
		resp := do(t, ts, http.MethodPost, "/api/v1/resolve", map[string]any{
			"dependencies": []map[string]string{selector},
		}, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var out struct {
			Entries []struct {
				Version string `json:"version"`
			} `json:"entries"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out.Entries[0].Version
	}

	if got := request(map[string]string{"name": "cabextract"}); got != "2.0" {
		t.Errorf("expected newest version 2.0, got %s", got)
	}
	if got := request(map[string]string{"name": "cabextract", "version": "1.0"}); got != "1.0" {
		t.Errorf("expected exact version 1.0, got %s", got)
	}
}

func TestResolve_RejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body any
	}{
		{name: "empty", body: map[string]any{}},
		{name: "multiple selectors", body: map[string]any{"dependencies": []map[string]string{{"name": "one", "id": uuid.NewString()}}}},
		{name: "invalid id", body: map[string]any{"dependencies": []map[string]string{{"id": "not-a-uuid"}}}},
		{name: "slot dependency", body: map[string]any{"dependencies": []map[string]string{{"slot": "runner"}}}},
		{name: "invalid slot", body: map[string]any{"components": []map[string]string{{"slot": "other"}}}},
		{name: "invalid platform", body: map[string]any{"dependencies": []map[string]string{{"name": "one"}}, "platform": map[string]string{"os": "android", "arch": "x86_64"}}},
		{name: "unknown field", body: map[string]any{"dependencies": []map[string]string{{"name": "one"}}, "extra": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, _ := setup(t)
			resp := do(t, ts, http.MethodPost, "/api/v1/resolve", tt.body, "")
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestResolve_RejectsMissingRequirementsAndCycles(t *testing.T) {
	t.Run("missing requirement", func(t *testing.T) {
		ts, s := setup(t)
		putBuiltEntry(t, s, models.KindDependency, "broken", "1", nil,
			[]models.Requirement{{Name: "missing"}}, []models.CatalogArtifact{resolveArtifact("broken.zip", nil)}, time.Unix(1, 0))
		resp := do(t, ts, http.MethodPost, "/api/v1/resolve", map[string]any{
			"dependencies": []map[string]string{{"name": "broken"}},
		}, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("cycle", func(t *testing.T) {
		ts, s := setup(t)
		idA, idB := uuid.NewString(), uuid.NewString()
		for _, dep := range []models.Dependency{
			{ID: idA, Name: "a", Kind: models.KindDependency, Status: "built", Entry: models.CatalogEntry{ID: idA, Name: "a", Version: "1", Requirements: []models.Requirement{{ID: idB}}, Artifacts: []models.CatalogArtifact{resolveArtifact("a.zip", nil)}}},
			{ID: idB, Name: "b", Kind: models.KindDependency, Status: "built", Entry: models.CatalogEntry{ID: idB, Name: "b", Version: "1", Requirements: []models.Requirement{{ID: idA}}, Artifacts: []models.CatalogArtifact{resolveArtifact("b.zip", nil)}}},
		} {
			if err := s.Deps.Put(context.Background(), dep.ID, dep, 0); err != nil {
				t.Fatal(err)
			}
		}
		resp := do(t, ts, http.MethodPost, "/api/v1/resolve", map[string]any{
			"dependencies": []map[string]string{{"id": idA}},
		}, "")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}

func TestDownloadBatch_DeduplicatesArtifactsAndTracksSources(t *testing.T) {
	ts, s := setup(t)
	artifact := resolveArtifact("shared.zip", nil)
	putBuiltEntry(t, s, models.KindDependency, "one", "1", nil, nil, []models.CatalogArtifact{artifact}, time.Unix(1, 0))
	putBuiltEntry(t, s, models.KindDependency, "two", "1", nil, nil, []models.CatalogArtifact{artifact}, time.Unix(2, 0))

	resp := do(t, ts, http.MethodPost, "/api/v1/download-batch", map[string]any{
		"dependencies": []map[string]string{{"name": "one"}, {"name": "two"}},
	}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out struct {
		Downloads []struct {
			FileName string `json:"file_name"`
			Sources  []struct {
				Name string `json:"name"`
			} `json:"sources"`
		} `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Downloads) != 1 {
		t.Fatalf("expected 1 download, got %d", len(out.Downloads))
	}
	if out.Downloads[0].FileName != "shared.zip" || len(out.Downloads[0].Sources) != 2 {
		t.Errorf("unexpected download: %+v", out.Downloads[0])
	}
}
