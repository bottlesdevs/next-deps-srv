package models

import "time"

type Dependency struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending_review, approved, building, built, rejected
	SubmittedBy string    `json:"submitted_by"`
	ReviewedBy  string    `json:"reviewed_by,omitempty"`
	ReviewNote   string    `json:"review_note,omitempty"`
	RejectReason string    `json:"reject_reason,omitempty"`
	Manifest    Manifest  `json:"manifest"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Manifest struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	ExpectedHash string        `json:"expected_hash"`
	License      string        `json:"license"`
	LicenseFiles []LicenseFile `json:"license_files"`
}

type LicenseFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
