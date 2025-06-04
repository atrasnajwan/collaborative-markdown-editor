package document

import (
	"time"
)

type Document struct {
    ID        uint      `json:"id"`
    Name      string    `json:"name"`
    Content   *string   `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    UserID    uint      `json:"user_id"`
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
