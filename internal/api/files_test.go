package api_test

import (
"context"
"net/http"
"testing"
"time"

"github.com/bottlesdevs/next-deps-srv/internal/models"
)

func TestGetFile_NotFound(t *testing.T) {
ts, _ := setup(t)
resp, err := http.Get(ts.URL + "/api/v1/files/nonexistent.dll")
if err != nil {
t.Fatal(err)
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusNotFound {
t.Errorf("expected 404, got %d", resp.StatusCode)
}
}

func TestGetFile_WithRevisions(t *testing.T) {
ts, s := setup(t)
ctx := context.Background()

file, _ := s.CreateFile(ctx, models.BucketFile{
Name:       "kernel32.dll",
BucketChar: "k",
CreatedAt:  time.Now(),
})

rev, _ := s.CreateRevision(ctx, models.FileRevision{
FileID:      file.ID,
RevisionNum: 1,
Hash:        "abc123",
SourceJobID: "job-1",
SourceDepID: "dep-1",
StoragePath: "/noop/kernel32.dll",
SizeBytes:   1024,
CreatedAt:   time.Now(),
})

file.LatestRevID = rev.ID
s.UpdateFile(ctx, file)

resp, err := http.Get(ts.URL + "/api/v1/files/kernel32.dll")
if err != nil {
t.Fatal(err)
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
t.Errorf("expected 200, got %d", resp.StatusCode)
}
}
