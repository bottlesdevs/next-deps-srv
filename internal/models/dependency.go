package models

import (
	"encoding/json"
	"time"
)

// SchemaVersion is the version emitted in the published catalog documents.
const SchemaVersion = 1

// Kind is the addon role of an entry. Dependencies are published in the
// dependency catalog and every other role in the component catalog.
type Kind string

const (
	KindRunner     Kind = "runner"
	KindWinebridge Kind = "winebridge"
	KindUmu        Kind = "umu"
	KindComponent  Kind = "component"
	KindDependency Kind = "dependency"
)

var Kinds = []Kind{KindRunner, KindWinebridge, KindUmu, KindComponent, KindDependency}

// Slot is the mutually exclusive prefix position a component occupies.
type Slot string

const (
	SlotDXVK        Slot = "dxvk"
	SlotVKD3D       Slot = "vkd3d"
	SlotNVAPI       Slot = "nvapi"
	SlotLatencyFlex Slot = "latency-flex"
)

var Slots = []Slot{SlotDXVK, SlotVKD3D, SlotNVAPI, SlotLatencyFlex}

// Flavor is how a runner is launched.
type Flavor string

const (
	FlavorWine   Flavor = "wine"
	FlavorProton Flavor = "proton"
)

var Flavors = []Flavor{FlavorWine, FlavorProton}

type OperatingSystem string

const (
	OSLinux   OperatingSystem = "linux"
	OSMacOS   OperatingSystem = "mac-os"
	OSWindows OperatingSystem = "windows"
)

var OperatingSystems = []OperatingSystem{OSLinux, OSMacOS, OSWindows}

type Architecture string

const (
	ArchX86     Architecture = "x86"
	ArchX86_64  Architecture = "x86_64"
	ArchAArch64 Architecture = "aarch64"
)

var Architectures = []Architecture{ArchX86, ArchX86_64, ArchAArch64}

// ChecksumAlgorithms are the only digests the catalog schema admits.
var ChecksumAlgorithms = []string{"sha256", "sha512"}

type Dependency struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Category     string       `json:"category"`
	Description  string       `json:"description"`
	License      string       `json:"license,omitempty"`
	Status       string       `json:"status"` // pending_review, approved, building, built, rejected
	SubmittedBy  string       `json:"submitted_by"`
	ReviewedBy   string       `json:"reviewed_by,omitempty"`
	ReviewNote   string       `json:"review_note,omitempty"`
	RejectReason string       `json:"reject_reason,omitempty"`
	Entry        CatalogEntry `json:"entry"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Catalog is one published catalog document. The component and dependency
// catalogs share this shape.
type Catalog struct {
	SchemaVersion int            `json:"schema_version"`
	Entries       []CatalogEntry `json:"entries"`
}

// CatalogEntry is a release advertised by the catalog.
type CatalogEntry struct {
	ID           string            `json:"id"` // uuid
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Kind         Kind              `json:"kind"`
	Slot         *Slot             `json:"slot,omitempty"`   // components only
	Flavor       *Flavor           `json:"flavor,omitempty"` // runners only
	Artifacts    []CatalogArtifact `json:"artifacts"`
	Requirements []Requirement     `json:"requirements,omitempty"`
	// Settings and Effects are opaque to the server and carried through to
	// consumers untouched.
	Settings []json.RawMessage `json:"settings,omitempty"`
	Effects  []json.RawMessage `json:"effects,omitempty"`
}

// CatalogArtifact is one downloadable file and the recipe associated with it.
type CatalogArtifact struct {
	URL      string   `json:"url"`
	FileName string   `json:"file_name"`
	Checksum Checksum `json:"checksum"`
	Platform *Target  `json:"platform,omitempty"`
	// Steps is an opaque recipe carried through to consumers untouched.
	Steps []json.RawMessage `json:"steps,omitempty"`
}

// Checksum is the expected digest of a downloaded artifact. Verification
// compares the value exactly and case-sensitively against a lowercase
// hexadecimal digest.
type Checksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

// Target is the host operating-system and architecture pair an artifact is
// built for. Matching is exact; no compatibility is inferred.
type Target struct {
	OS   OperatingSystem `json:"os"`
	Arch Architecture    `json:"arch"`
}

// String renders a target as "os/arch", or an empty string when unset.
func (t *Target) String() string {
	if t == nil {
		return ""
	}
	return string(t.OS) + "/" + string(t.Arch)
}

// Requirement is a constraint satisfied by another addon in the prefix.
// Exactly one field is set: name and id may be satisfied by an addon of any
// kind, slot only by a component.
type Requirement struct {
	Name string `json:"name,omitempty"`
	Slot *Slot  `json:"slot,omitempty"`
	ID   string `json:"id,omitempty"`
}

// NormalizeForCatalog guarantees the slices the schema types as arrays are
// non-nil, so they marshal as [] rather than null.
func (e CatalogEntry) NormalizeForCatalog() CatalogEntry {
	if e.Artifacts == nil {
		e.Artifacts = []CatalogArtifact{}
	}
	return e
}
