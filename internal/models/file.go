package models

import "time"

type BucketFile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	BucketChar  string    `json:"bucket_char"`
	LatestRevID string    `json:"latest_rev_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FileRevision struct {
	ID          string    `json:"id"`
	FileID      string    `json:"file_id"`
	RevisionNum int       `json:"revision_num"`
	Hash        string    `json:"hash"`
	SourceJobID string    `json:"source_job_id"`
	SourceDepID string    `json:"source_dep_id"`
	ArchiveURL  string    `json:"archive_url"`
	ArchiveHash string    `json:"archive_hash"`
	StoragePath string    `json:"storage_path"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}
