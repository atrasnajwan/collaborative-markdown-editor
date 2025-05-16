package user

import (
	"collaborative-markdown-editor/internal/document"
	"time"
)

type User struct {
	ID           uint
	Name         string
	Email        string `gorm:"uniqueIndex"`
	PasswordHash uint8
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsActive     bool `gorm:"default:true"`
	Documents    []document.Document
}
