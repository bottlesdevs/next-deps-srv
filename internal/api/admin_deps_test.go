package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

func TestAdminDeleteDep_PreservesIndexedFiles(t *testing.T) {
	ts, s := setup(t)
	ctx := context.Background()
	token := createAdminToken(t, s)

	entry := testEntry("obsolete").CatalogEntry
	entry.ID = uuid.NewString()
	dep, err := s.CreateDep(ctx, models.Dependency{
		ID: entry.ID, Name: entry.Name, Kind: models.KindDependency,
		Status: "built", SubmittedBy: "user-1", Entry: entry,
	})
	if err != nil {
		t.Fatal(err)
	}
	file, err := s.CreateFile(ctx, models.BucketFile{Name: "obsolete.dll", BucketChar: "o"})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := s.CreateRevision(ctx, models.FileRevision{
		FileID: file.ID, RevisionNum: 1, Hash: "hash",
		SourceDepID: dep.ID, StoragePath: "o/obsolete.dll/1",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp := do(t, ts, http.MethodDelete, "/api/v1/admin/deps/"+dep.ID, nil, token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if _, err := s.GetDep(ctx, dep.ID); err == nil {
		t.Error("expected dependency reference to be deleted")
	}
	if _, err := s.GetFile(ctx, file.ID); err != nil {
		t.Errorf("bucket file was deleted: %v", err)
	}
	if _, err := s.GetRevision(ctx, revision.ID); err != nil {
		t.Errorf("file revision was deleted: %v", err)
	}
}

func TestAdminDeleteDep_RequiresAdmin(t *testing.T) {
	ts, s := setup(t)
	entry := testEntry("protected").CatalogEntry
	entry.ID = uuid.NewString()
	dep, err := s.CreateDep(context.Background(), models.Dependency{
		ID: entry.ID, Name: entry.Name, Kind: models.KindDependency,
		Status: "built", SubmittedBy: "user-1", Entry: entry,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp := do(t, ts, http.MethodDelete, "/api/v1/admin/deps/"+dep.ID, nil, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if _, err := s.GetDep(context.Background(), dep.ID); err != nil {
		t.Errorf("dependency was deleted without admin auth: %v", err)
	}
}
