package document

import (
	"time"

	"gorm.io/gorm"
)

type DocumentsMeta struct {
	Total       int64 `json:"total"`
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalPage   int   `json:"total_page"`
}

type DocumentsData struct {
	Documents []Document
	Meta      DocumentsMeta `json:"total_page"`
}
type DocumentRepository interface {
	Create(userID uint64, document *Document) error
	UpdateContent(id uint64, userID uint64, content []byte) error
	GetByUserID(userID uint64, page, pageSize int) (DocumentsData, error)
	FindByID(id uint64) (*Document, error)
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

func (r *DocumentRepositoryImpl) UpdateContent(id uint64, userId uint64, content []byte) error {
    var seq uint64

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. increment sequence on document
		if err := tx.Raw(`
			UPDATE documents
			SET update_seq = update_seq + 1,
			    updated_at = NOW()
			WHERE id = ?
			RETURNING update_seq
		`, documentID).Scan(&seq).Error; err != nil {
			return err
		}

		// 2. Insert document update with the generated seq
		if err := tx.Create(&DocumentUpdate{
			DocumentID:   	documentID,
			Seq:          	seq,
			UpdateBinary: 	content,
			UserID:       	userID,
			CreatedAt: 		time.Now().UTC(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

    return err
}

func (r *DocumentRepositoryImpl) GetByUserID(userID uint64, page, pageSize int) (DocumentsData, error) {
	var documents []Document
	var totalRecords int64

	// Count total records
	if err := r.db.Model(&Document{}).Where("user_id = ?", userID).Count(&totalRecords).Error; err != nil {
		return DocumentsData{}, err
	}

	offset := (page - 1) * pageSize
	err := r.db.Where("user_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Find(&documents).Error

	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))

	return DocumentsData{
		Documents: documents,
		Meta: DocumentsMeta{
			Total:       totalRecords,
			PerPage:     pageSize,
			TotalPage:   totalPages,
			CurrentPage: page,
		},
	}, err
}

func (r *DocumentRepositoryImpl) FindByID(id uint64) (*Document, error) {
	var doc Document
	err := r.db.First(&doc, id).Error
	return &doc, err
}