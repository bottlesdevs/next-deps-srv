package models

import "time"

type BuildJob struct {
	ID           string    `json:"id"`
	DependencyID string    `json:"dependency_id"`
	Status       string    `json:"status"` // queued, running, done, failed
	Error        string    `json:"error,omitempty"`
	FilesIndexed int       `json:"files_indexed"`
	Logs         []string  `json:"logs,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
}
