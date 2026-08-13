package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/models"
	"github.com/google/uuid"
)

const (
	sha256Hex = "b50dc50ec7f41d58b115a6b685d4d1315ba3c797bd3aa0f49213f2703cb82388"
	sha512Hex = "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce" +
		"47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
)

func validEntry() models.CatalogEntry {
	return models.CatalogEntry{
		ID:      uuid.NewString(),
		Name:    "Wine Stable",
		Version: "11.0",
		Artifacts: []models.CatalogArtifact{{
			URL:      "https://example.com/wine-osx64.tar.xz",
			FileName: "wine-osx64.tar.xz",
			Checksum: models.Checksum{Algorithm: "sha256", Value: sha256Hex},
			Platform: &models.Target{OS: models.OSMacOS, Arch: models.ArchAArch64},
		}},
	}
}

func componentEntry() models.CatalogEntry {
	e := validEntry()
	slot := models.SlotRunner
	e.Slot = &slot
	return e
}

func TestValidate_DependencyAndComponent(t *testing.T) {
	if err := validEntry().Validate(models.KindDependency); err != nil {
		t.Errorf("valid dependency rejected: %v", err)
	}
	if err := componentEntry().Validate(models.KindComponent); err != nil {
		t.Errorf("valid component rejected: %v", err)
	}
}

// Slot is the only structural difference between the two catalogs.
func TestValidate_SlotBelongsToComponentsOnly(t *testing.T) {
	if err := componentEntry().Validate(models.KindDependency); err == nil {
		t.Error("expected a dependency carrying a slot to be rejected")
	}
	if err := validEntry().Validate(models.KindComponent); err == nil {
		t.Error("expected a component without a slot to be rejected")
	}
	e := componentEntry()
	bad := models.Slot("gpu")
	e.Slot = &bad
	if err := e.Validate(models.KindComponent); err == nil {
		t.Error("expected an unknown slot to be rejected")
	}
}

func TestValidate_Rejects(t *testing.T) {
	tests := map[string]func(*models.CatalogEntry){
		"non-uuid id":      func(e *models.CatalogEntry) { e.ID = "openssl" },
		"missing name":     func(e *models.CatalogEntry) { e.Name = "" },
		"missing version":  func(e *models.CatalogEntry) { e.Version = "" },
		"no artifacts":     func(e *models.CatalogEntry) { e.Artifacts = nil },
		"missing url":      func(e *models.CatalogEntry) { e.Artifacts[0].URL = "" },
		"missing filename": func(e *models.CatalogEntry) { e.Artifacts[0].FileName = "" },
		"no checksum": func(e *models.CatalogEntry) {
			e.Artifacts[0].Checksum = models.Checksum{}
		},
		"md5 no longer allowed": func(e *models.CatalogEntry) {
			e.Artifacts[0].Checksum = models.Checksum{Algorithm: "md5", Value: strings.Repeat("a", 32)}
		},
		"sha1 no longer allowed": func(e *models.CatalogEntry) {
			e.Artifacts[0].Checksum = models.Checksum{Algorithm: "sha1", Value: strings.Repeat("a", 40)}
		},
		"uppercase digest": func(e *models.CatalogEntry) {
			e.Artifacts[0].Checksum.Value = strings.ToUpper(sha256Hex)
		},
		"wrong digest length": func(e *models.CatalogEntry) {
			e.Artifacts[0].Checksum.Value = sha256Hex[:40]
		},
		"unknown os": func(e *models.CatalogEntry) {
			e.Artifacts[0].Platform = &models.Target{OS: "freebsd", Arch: models.ArchX86_64}
		},
		"unknown arch": func(e *models.CatalogEntry) {
			e.Artifacts[0].Platform = &models.Target{OS: models.OSLinux, Arch: "riscv"}
		},
		"empty requirement": func(e *models.CatalogEntry) {
			e.Requirements = []models.Requirement{{}}
		},
		"requirement with two variants": func(e *models.CatalogEntry) {
			slot := models.SlotRunner
			e.Requirements = []models.Requirement{{Name: "wine", Slot: &slot}}
		},
		"requirement id not a uuid": func(e *models.CatalogEntry) {
			e.Requirements = []models.Requirement{{ID: "not-a-uuid"}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := validEntry()
			mutate(&e)
			if err := e.Validate(models.KindDependency); err == nil {
				t.Errorf("expected %s to be rejected", name)
			}
		})
	}
}

func TestValidate_AcceptsSha512AndRequirements(t *testing.T) {
	e := validEntry()
	e.Artifacts[0].Checksum = models.Checksum{Algorithm: "sha512", Value: sha512Hex}
	slot := models.SlotRunner
	e.Requirements = []models.Requirement{
		{Name: "wine"},
		{Slot: &slot},
		{ID: uuid.NewString()},
	}
	if err := e.Validate(models.KindDependency); err != nil {
		t.Errorf("expected entry to be valid, got %v", err)
	}
}

func TestValidate_AllowsSameFileDifferentPlatform(t *testing.T) {
	e := validEntry()
	second := e.Artifacts[0]
	second.Platform = &models.Target{OS: models.OSLinux, Arch: models.ArchX86_64}
	e.Artifacts = append(e.Artifacts, second)
	if err := e.Validate(models.KindDependency); err != nil {
		t.Errorf("expected per-platform artifacts to be allowed, got %v", err)
	}
}

func TestValidate_RejectsDuplicateArtifacts(t *testing.T) {
	e := validEntry()
	e.Artifacts = append(e.Artifacts, e.Artifacts[0])
	if err := e.Validate(models.KindDependency); err == nil {
		t.Error("expected duplicate artifacts to be rejected")
	}
}

// The schema sets additionalProperties:false, so absent optional members must
// be omitted rather than emitted empty, and required arrays must never be null.
func TestCatalogJSON_Shape(t *testing.T) {
	doc := models.Catalog{
		SchemaVersion: models.SchemaVersion,
		Entries:       []models.CatalogEntry{validEntry().NormalizeForCatalog()},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, absent := range []string{`"slot"`, `"requirements"`, `"steps"`, `"size"`, `"component_root"`} {
		if strings.Contains(got, absent) {
			t.Errorf("expected %s to be omitted, got %s", absent, got)
		}
	}
	for _, present := range []string{`"schema_version":1`, `"entries":[`, `"checksum":{`, `"platform":{`} {
		if !strings.Contains(got, present) {
			t.Errorf("expected %s in %s", present, got)
		}
	}

	empty := models.Catalog{SchemaVersion: models.SchemaVersion, Entries: []models.CatalogEntry{}}
	b, _ = json.Marshal(empty)
	if !strings.Contains(string(b), `"entries":[]`) {
		t.Errorf("empty catalog must emit [] not null, got %s", b)
	}

	// An entry with no artifacts must still marshal the required array.
	bare := models.CatalogEntry{ID: uuid.NewString(), Name: "x", Version: "1"}.NormalizeForCatalog()
	b, _ = json.Marshal(bare)
	if !strings.Contains(string(b), `"artifacts":[]`) {
		t.Errorf("artifacts must marshal as [] not null, got %s", b)
	}
}

// Steps are opaque and must survive a round trip untouched.
func TestSteps_RoundTripUnchanged(t *testing.T) {
	raw := `{"schema_version":1,"entries":[{"id":"` + uuid.NewString() + `","name":"n","version":"1","artifacts":[{"url":"https://e.com/a.zip","file_name":"a.zip","checksum":{"algorithm":"sha256","value":"` + sha256Hex + `"},"steps":[{"extract":{"strip":1}},"noop",42]}]}]}`
	var doc models.Catalog
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"steps":[{"extract":{"strip":1}},"noop",42]`) {
		t.Errorf("steps were altered in round trip: %s", out)
	}
}
