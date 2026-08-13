package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/bottlesdevs/next-deps-srv/internal/store"
	"github.com/google/uuid"
)

func testStoreAndBackend(t *testing.T) (*store.Store, bucket.Backend) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(s.Close)

	b, err := bucket.NewLocalBackend(models.LocalStorageConfig{
		BucketRoot: filepath.Join(dir, "bucket"),
		DedupRoot:  filepath.Join(dir, "dedup"),
	})
	if err != nil {
		t.Fatalf("NewLocalBackend: %v", err)
	}
	return s, b
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func newDep(t *testing.T, s *store.Store, name string) models.Dependency {
	t.Helper()
	id := uuid.NewString()
	d, err := s.CreateDep(context.Background(), models.Dependency{
		ID: id, Name: name, Kind: models.KindDependency, Status: "approved",
		Entry: models.CatalogEntry{ID: id, Name: name, Version: "1.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Two dependencies producing byte-identical content must each end up with the
// file attributed to them. Deduplicating on content alone left the second
// build with no revisions, so its indexed files came back empty.
func TestIndexOneFile_AttributesSharedContentToEachDep(t *testing.T) {
	ctx := context.Background()
	s, backend := testStoreAndBackend(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "shared.dll", "identical bytes")

	depA := newDep(t, s, "alpha")
	depB := newDep(t, s, "beta")
	art := models.CatalogArtifact{URL: "https://e.com/a.zip", FileName: "a.zip"}

	actionA, err := indexOneFile(ctx, src, "shared.dll", models.BuildJob{ID: "job-a"}, depA, art, "archive-a", s, backend)
	if err != nil {
		t.Fatal(err)
	}
	if actionA != "new" {
		t.Errorf("first index should be new, got %q", actionA)
	}

	actionB, err := indexOneFile(ctx, src, "shared.dll", models.BuildJob{ID: "job-b"}, depB, art, "archive-b", s, backend)
	if err != nil {
		t.Fatal(err)
	}
	if actionB == "skip – same hash" {
		t.Fatal("second dependency was skipped, so its files would be invisible")
	}

	for _, d := range []models.Dependency{depA, depB} {
		revs, err := s.RevisionsByDep(ctx, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(revs) != 1 {
			t.Errorf("%s: expected 1 revision attributed, got %d", d.Name, len(revs))
		}
	}

	// The bytes must not be stored twice: the second revision shares storage.
	revsA, _ := s.RevisionsByDep(ctx, depA.ID)
	revsB, _ := s.RevisionsByDep(ctx, depB.ID)
	if revsA[0].StoragePath != revsB[0].StoragePath {
		t.Errorf("expected shared storage, got %q and %q", revsA[0].StoragePath, revsB[0].StoragePath)
	}
}

// Re-running the same dependency must stay idempotent.
func TestIndexOneFile_SkipsRebuildOfSameDep(t *testing.T) {
	ctx := context.Background()
	s, backend := testStoreAndBackend(t)
	dir := t.TempDir()
	src := writeFile(t, dir, "same.dll", "unchanged")

	dep := newDep(t, s, "alpha")
	art := models.CatalogArtifact{URL: "https://e.com/a.zip", FileName: "a.zip"}

	if _, err := indexOneFile(ctx, src, "same.dll", models.BuildJob{ID: "job-1"}, dep, art, "h", s, backend); err != nil {
		t.Fatal(err)
	}
	action, err := indexOneFile(ctx, src, "same.dll", models.BuildJob{ID: "job-2"}, dep, art, "h", s, backend)
	if err != nil {
		t.Fatal(err)
	}
	if action != "skip – same hash" {
		t.Errorf("expected a rebuild to skip, got %q", action)
	}
	revs, _ := s.RevisionsByDep(ctx, dep.ID)
	if len(revs) != 1 {
		t.Errorf("rebuild should not add revisions, got %d", len(revs))
	}
}

// Changed content still produces a new revision for the same dependency.
func TestIndexOneFile_NewRevisionOnChangedContent(t *testing.T) {
	ctx := context.Background()
	s, backend := testStoreAndBackend(t)
	dir := t.TempDir()
	dep := newDep(t, s, "alpha")
	art := models.CatalogArtifact{URL: "https://e.com/a.zip", FileName: "a.zip"}

	src := writeFile(t, dir, "changing.dll", "v1")
	if _, err := indexOneFile(ctx, src, "changing.dll", models.BuildJob{ID: "job-1"}, dep, art, "h1", s, backend); err != nil {
		t.Fatal(err)
	}
	src2 := writeFile(t, filepath.Join(dir), "changing2.dll", "v2")
	if _, err := indexOneFile(ctx, src2, "changing.dll", models.BuildJob{ID: "job-2"}, dep, art, "h2", s, backend); err != nil {
		t.Fatal(err)
	}
	revs, _ := s.RevisionsByDep(ctx, dep.ID)
	if len(revs) != 2 {
		t.Errorf("expected 2 revisions for changed content, got %d", len(revs))
	}
}
