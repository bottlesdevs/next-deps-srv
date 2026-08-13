package models

import "time"

// SchemaVersion is the version emitted in the published catalog document.
const SchemaVersion = 1

type Dependency struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Description  string    `json:"description"`
	License      string    `json:"license,omitempty"`
	Status       string    `json:"status"` // pending_review, approved, building, built, rejected
	SubmittedBy  string    `json:"submitted_by"`
	ReviewedBy   string    `json:"reviewed_by,omitempty"`
	ReviewNote   string    `json:"review_note,omitempty"`
	RejectReason string    `json:"reject_reason,omitempty"`
	Item         Item      `json:"item"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Catalog is the document consumers fetch. It mirrors the published schema:
// a schema_version plus the list of items.
type Catalog struct {
	SchemaVersion int    `json:"schema_version"`
	Items         []Item `json:"items"`
}

// Item is one entry in the catalog document.
type Item struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Kind      *Kind      `json:"kind,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
}

type Kind struct {
	Type    string `json:"type"`
	Flavour string `json:"flavour"`
}

type Artifact struct {
	URL           string    `json:"url"`
	FileName      string    `json:"file_name"`
	Checksum      *Checksum `json:"checksum,omitempty"`
	Size          int64     `json:"size"`
	Platform      *Platform `json:"platform,omitempty"`
	ComponentRoot string    `json:"component_root"`
}

type Checksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// String renders a platform as "os/arch", or an empty string when unset.
func (p *Platform) String() string {
	if p == nil {
		return ""
	}
	return p.OS + "/" + p.Arch
}
