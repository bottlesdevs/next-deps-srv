package models

import "time"

type CommunityPost struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	ParentID  string    `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Deleted   bool      `json:"deleted"`
}
