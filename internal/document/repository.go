package document

import (
	"time"

	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(userID uint, document *Document) error
}

// UserRepositoryImpl implements User
type DocumentRepositoryImpl struct{
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
