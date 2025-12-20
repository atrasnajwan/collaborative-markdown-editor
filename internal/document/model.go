package document


import (
	"time"
)


type Document struct {
	ID            uint64    `gorm:"primaryKey;auto"`
	Title         string    `gorm:"type:text;not null"`
	UserID   	  uint64    `gorm:"not null;index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Updates       []DocumentUpdate   `gorm:"constraint:OnDelete:CASCADE"`
	Snapshots     []DocumentSnapshot `gorm:"constraint:OnDelete:CASCADE"`
	Versions      []DocumentVersion  `gorm:"constraint:OnDelete:CASCADE"`
	Collaborators []DocumentCollaborator `gorm:"constraint:OnDelete:CASCADE"`
}

type DocumentUpdate struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	DocumentID    uint64    `gorm:"not null;index;index:idx_doc_seq,priority:1"`
	Seq           uint64    `gorm:"not null;index:idx_doc_seq,priority:2"`
	UpdateBinary  []byte    `gorm:"type:bytea;not null"`
	UserID        uint64    `gorm:"not null;index"`
	// ClientID      string    `gorm:"type:text"`
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
	CreatedAt  time.Time
}

type DocumentCollaborator struct {
	DocumentID uint64 	`gorm:"primaryKey"`
	UserID     uint64	`gorm:"primaryKey"`
	Role       string    `gorm:"type:text;not null"`
	AddedAt    time.Time
}
