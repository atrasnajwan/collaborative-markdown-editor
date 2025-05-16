package user

import (
	"collaborative-markdown-editor/internal/document"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           uint
	Name         string
	Email        string `gorm:"uniqueIndex"`
	Password     string `gorm:"-"` // input only, not on db
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IsActive     bool `gorm:"default:true"`
	Documents    []document.Document
}

// BeforeCreate GORM hook: hash the password before saving
func (user *User) BeforeCreate(db *gorm.DB) (err error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashed)
	return nil
}

func (user *User) Deactivate(db *gorm.DB) error {
	return db.Model(user).Update("is_active", false).Error
}