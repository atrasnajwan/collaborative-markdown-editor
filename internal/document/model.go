package document

import (
	"time"
)

type Document struct {
	ID        uint
	Name      string
	Content   *string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    uint
}

type DocumentPermission struct {
	ID          uint
	DocumentID  uint
	Document    Document
	UserID      uint 
	GrantedByID uint
	AccessLevel string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DocumentEdit struct {
	ID            uint
	DocumentID    uint
	Document      Document
	UserID        uint
	Position      uint
	DeletedLength *uint
	InsertContent *string
	UpdatedAt     time.Time
}
