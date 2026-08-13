package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
)

func validItem() models.Item {
	return models.Item{
		ID:      "openssl",
		Name:    "OpenSSL",
		Version: "3.0.0",
		Kind:    &models.Kind{Type: "library", Flavour: "shared"},
		Artifacts: []models.Artifact{{
			URL:           "https://example.com/openssl-win64.zip",
			FileName:      "openssl-win64.zip",
			Checksum:      &models.Checksum{Algorithm: "sha256", Value: "abc123"},
			Size:          2048,
			Platform:      &models.Platform{OS: "windows", Arch: "x86_64"},
			ComponentRoot: "openssl",
		}},
	}
}

func TestItemValidate_Valid(t *testing.T) {
	if err := validItem().Validate(); err != nil {
		t.Fatalf("expected valid item, got %v", err)
	}
}

func TestItemValidate_Rejects(t *testing.T) {
	tests := map[string]func(*models.Item){
		"missing id":             func(i *models.Item) { i.ID = "" },
		"missing name":           func(i *models.Item) { i.Name = "" },
		"missing version":        func(i *models.Item) { i.Version = "" },
		"no artifacts":           func(i *models.Item) { i.Artifacts = nil },
		"partial kind":           func(i *models.Item) { i.Kind = &models.Kind{Type: "library"} },
		"missing url":            func(i *models.Item) { i.Artifacts[0].URL = "" },
		"missing file_name":      func(i *models.Item) { i.Artifacts[0].FileName = "" },
		"missing component_root": func(i *models.Item) { i.Artifacts[0].ComponentRoot = "" },
		"negative size":          func(i *models.Item) { i.Artifacts[0].Size = -1 },
		"partial platform":       func(i *models.Item) { i.Artifacts[0].Platform = &models.Platform{OS: "windows"} },
		"empty checksum value": func(i *models.Item) {
			i.Artifacts[0].Checksum = &models.Checksum{Algorithm: "sha256"}
		},
		"unsupported algorithm": func(i *models.Item) {
			i.Artifacts[0].Checksum = &models.Checksum{Algorithm: "crc32", Value: "abc"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			item := validItem()
			mutate(&item)
			if err := item.Validate(); err == nil {
				t.Errorf("expected %s to be rejected", name)
			}
		})
	}
}

func TestItemValidate_RejectsDuplicateArtifacts(t *testing.T) {
	item := validItem()
	item.Artifacts = append(item.Artifacts, item.Artifacts[0])
	if err := item.Validate(); err == nil {
		t.Error("expected duplicate artifacts to be rejected")
	}
}

func TestItemValidate_AllowsSameFileDifferentPlatform(t *testing.T) {
	item := validItem()
	second := item.Artifacts[0]
	second.Platform = &models.Platform{OS: "windows", Arch: "x86"}
	item.Artifacts = append(item.Artifacts, second)
	if err := item.Validate(); err != nil {
		t.Errorf("expected per-platform artifacts to be allowed, got %v", err)
	}
}

// Optional members must be omitted, not emitted empty, so the published
// document keeps satisfying the schema's minLength constraints.
func TestCatalogJSON_OmitsOptionalMembers(t *testing.T) {
	doc := models.Catalog{
		SchemaVersion: models.SchemaVersion,
		Items: []models.Item{{
			ID:      "zlib",
			Name:    "zlib",
			Version: "1.3",
			Artifacts: []models.Artifact{{
				URL:           "https://example.com/zlib.zip",
				FileName:      "zlib.zip",
				Size:          10,
				ComponentRoot: "zlib",
			}},
		}},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, absent := range []string{`"kind"`, `"checksum"`, `"platform"`} {
		if strings.Contains(got, absent) {
			t.Errorf("expected %s to be omitted, got %s", absent, got)
		}
	}
	for _, present := range []string{`"schema_version":1`, `"component_root":"zlib"`, `"size":10`} {
		if !strings.Contains(got, present) {
			t.Errorf("expected %s in %s", present, got)
		}
	}
}
