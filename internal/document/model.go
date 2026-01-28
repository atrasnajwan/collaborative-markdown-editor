package document


import (
	"time"
)


type Document struct {
	ID            uint64    `gorm:"primaryKey;auto" json:"id"`
	Title         string    `gorm:"type:text;not null" json:"title"`
	UserID   	  uint64    `gorm:"not null;index" json:"user_id"`
	UpdateSeq 	  uint64 	`gorm:"not null;default:0"`
	
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"` 

	Updates       []DocumentUpdate   `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Snapshots     []DocumentSnapshot `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Versions      []DocumentVersion  `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Collaborators []DocumentCollaborator `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

type DocumentUpdate struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	DocumentID    uint64    `gorm:"not null;index;index:idx_doc_seq,priority:1"`
	Seq           uint64    `gorm:"not null;index:idx_doc_seq,priority:2"`
	UpdateBinary  []byte    `gorm:"type:bytea;not null"`
	UserID        uint64    `gorm:"not null;index"`
	CreatedAt     time.Time
}

type DocumentSnapshot struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	DocumentID     uint64    `gorm:"not null;index"`
	Seq            uint64    `gorm:"not null;index"`
	SnapshotBinary []byte    `gorm:"type:bytea;not null"`
	CreatedAt      time.Time
}


type DocumentVersion struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	DocumentID  uint64    `gorm:"not null;index"`
	Name        string    `gorm:"type:text;not null"`
	Seq         uint64    `gorm:"not null"`
	CreatedBy  	uint64    `gorm:"not null;index"`
	CreatedAt  	time.Time
}

type DocumentCollaborator struct {
	DocumentID uint64 	`gorm:"primaryKey"`
	UserID     uint64	`gorm:"primaryKey"`
	Role       string    `gorm:"type:text;not null"`
	AddedAt    time.Time
}
