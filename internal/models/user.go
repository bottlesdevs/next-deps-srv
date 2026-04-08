package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Roles        []string  `json:"roles"` // admin, mod, contributor, viewer
	AvatarExt    string    `json:"avatar_ext,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	Website      string    `json:"website,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u User) GetID() string      { return u.ID }
func (u User) GetRoles() []string { return u.Roles }
