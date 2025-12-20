package user

import (
	"collaborative-markdown-editor/internal/document"
	"time"
)

// User represents a user in the system
type User struct {
	ID           uint64
	Name         string
	Email        string `gorm:"uniqueIndex"`
	Password     string `gorm:"-"` // input only, not stored in db
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsActive     bool `gorm:"default:true"`
	Documents    []document.Document
}

// SafeUser represents a user without sensitive information
type SafeUser struct {
	ID        uint64      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	IsActive  bool      `json:"is_active"`
}

// ToSafeUser converts a User to a SafeUser
func (u *User) ToSafeUser() SafeUser {
	return SafeUser{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		IsActive:  u.IsActive,
	}
}
