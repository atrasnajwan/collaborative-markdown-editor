package document

import (
	"context"
	"time"

	"gorm.io/gorm"
)


type DocumentRepository interface {
	Create(userID uint64, document *Document) error
	CreateUpdate(id uint64, userID uint64, content []byte) error
	GetByUserID(userID uint64, page, pageSize int) ([]Document, DocumentsMeta, error)
	GetUserRole(docID uint64, userID uint64) (string, error)
	FindByID(id uint64) (*Document, error)
	CurrentSeq(docID uint64, currentSeq *uint64) error
	CreateSnapshot(ctx context.Context, docID uint64, state []byte) error
	LastSnapshot(docID uint64, snapshot *DocumentSnapshot) error
	LastSnapshotSeq(docID uint64, lastSnapshotSeq *uint64) error
	UpdatesFromSnapshot(docID uint64, snapshotSeq uint64, updates *[]DocumentUpdate) error 
}

type DocumentRepositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new user repository
func NewRepository(db *gorm.DB) DocumentRepository {
	return &DocumentRepositoryImpl{db: db}
}

// Create creates a new user
func (r *DocumentRepositoryImpl) Create(userID uint64, document *Document) error {
	document.UserID = userID
	document.CreatedAt = time.Now().UTC() // Use UTC for consistency
	document.UpdatedAt = time.Now().UTC()
	document.Collaborators = []DocumentCollaborator{
		{
			UserID:  userID,
			Role:    "owner",
			AddedAt: time.Now().UTC(),
		},
	}
	return r.db.Create(document).Error
}

type DocumentsMeta struct {
	Total       int64 `json:"total"`
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalPage   int   `json:"total_page"`
}

func (r *DocumentRepositoryImpl) GetByUserID(userID uint64, page, pageSize int) ([]Document, DocumentsMeta, error) {
	var documents []Document
	var totalRecords int64

	// Count total records
	if err := r.db.Model(&Document{}).Where("user_id = ?", userID).Count(&totalRecords).Error; err != nil {
		return documents, DocumentsMeta{}, err
	}

	offset := (page - 1) * pageSize
	err := r.db.Where("user_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Find(&documents).Error

	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))

	return documents, DocumentsMeta{
			Total:       totalRecords,
			PerPage:     pageSize,
			TotalPage:   totalPages,
			CurrentPage: page,
		}, err
}

func (r *DocumentRepositoryImpl) FindByID(id uint64) (*Document, error) {
	var doc Document
	err := r.db.First(&doc, id).Error
	return &doc, err
}

func (r *DocumentRepositoryImpl) GetUserRole(docID uint64, userID uint64) (string, error) {
	var role string
	err := r.db.Model(&DocumentCollaborator{}).
				Where("document_id = ? AND user_id = ?", docID, userID).
				Select("role").
				Scan(&role).Error
	if err != nil || role == "" {
		return "none", err
	}

	return role, nil
}

func (r *DocumentRepositoryImpl) CreateUpdate(id uint64, userID uint64, content []byte) error {
    var seq uint64

	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		// 1. increment sequence on document
		if err := tx.Raw(`
			UPDATE documents
			SET update_seq = update_seq + 1,
			    updated_at = ?
			WHERE id = ?
			RETURNING update_seq
		`, now, id).Scan(&seq).Error; err != nil {
			return err
		}

		// 2. Insert document update with the generated seq
		if err := tx.Create(&DocumentUpdate{
			DocumentID:   	id,
			Seq:          	seq,
			UpdateBinary: 	content,
			UserID:       	userID,
			CreatedAt: 		now,
		}).Error; err != nil {
			return err
		}

		return nil
	})

    return err
}

func (r *DocumentRepositoryImpl) CreateSnapshot(ctx context.Context, docID uint64, state []byte) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var lastSeq uint64

		// Get latest update seq
		if err := tx.Model(&Document{}).
			Where("id = ?", docID).
			Select("update_seq").
			Scan(&lastSeq).Error; err != nil {
			return err
		}

		// check if already created
		var exists bool
		if err :=  tx.Model(&DocumentSnapshot{}).
			Select("count(1) > 0").
			Where("document_id = ? AND seq = ?", docID, lastSeq).
			Find(&exists).Error; err != nil {
			return err
		}

		if exists {
			return nil // snapshot already exists
		}
		
		// create snapshot
		snapshot := DocumentSnapshot{
			DocumentID: 	docID,
			Seq:    		lastSeq,
			SnapshotBinary:	state,
		}

		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}

		// cleanup old updates
		if err := tx.Where("document_id = ? AND seq <= ?", docID, lastSeq).
					 Delete(&DocumentUpdate{}).Error; err != nil {
			return err
		}

		return nil
	})
	return err
}

func (r *DocumentRepositoryImpl) CurrentSeq(docID uint64, currentSeq *uint64) error {
	return r.db.Model(&Document{}).
				Where("id = ?", docID).
				Select("update_seq").
				Scan(currentSeq).Error
}

func (r *DocumentRepositoryImpl) LastSnapshot(docID uint64, snapshot *DocumentSnapshot) error {
	return r.db.Where("document_id = ?", docID).
				Order("seq DESC").
				First(&snapshot).Error
}

func (r *DocumentRepositoryImpl) LastSnapshotSeq(docID uint64, lastSnapshotSeq *uint64) error {
	return r.db.Model(&DocumentSnapshot{}).
				Where("document_id = ?", docID).
				Select("COALESCE(MAX(seq), 0)").
				Scan(lastSnapshotSeq).Error
}

func (r *DocumentRepositoryImpl) UpdatesFromSnapshot(docID uint64, snapshotSeq uint64, updates *[]DocumentUpdate) error {
	return r.db.Where("document_id = ? AND seq > ?", docID, snapshotSeq).
				Order("seq ASC").
				Limit(500).
				Find(&updates).Error
}
