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
	Create(userID uint, document *Document) error
	GetByUserID(userID uint, page, pageSize int) (DocumentsData, error)
	FindByID(id uint) (*Document, error)
}

type DocumentRepositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new user repository
func NewRepository(db *gorm.DB) DocumentRepository {
	return &DocumentRepositoryImpl{db: db}
}

// Create creates a new user
func (r *DocumentRepositoryImpl) Create(userID uint, document *Document) error {
	document.UserID = userID
	document.CreatedAt = time.Now()
	document.UpdatedAt = time.Now()

	return r.db.Create(document).Error
}

func (r *DocumentRepositoryImpl) GetByUserID(userID uint, page, pageSize int) (DocumentsData, error) {
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

func (r *DocumentRepositoryImpl) FindByID(id uint) (*Document, error) {
	var doc Document
	err := r.db.First(&doc, id).Error
	return &doc, err
}