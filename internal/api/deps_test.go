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

func submitDepReq(t *testing.T, ts *httptest.Server, token string, manifest models.Manifest) *http.Response {
t.Helper()
b, _ := json.Marshal(manifest)
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
resp := submitDepReq(t, ts, "", models.Manifest{Name: "test", URL: "http://x.com", ExpectedHash: "abc"})
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

resp := submitDepReq(t, ts, token, models.Manifest{
Name:         "mylib",
URL:          "https://example.com/mylib.zip",
ExpectedHash: "deadbeef",
License:      "MIT",
})
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
Manifest:    models.Manifest{Name: "testdep", URL: "http://x.com", ExpectedHash: "abc"},
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
