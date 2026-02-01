package document

import (
	"collaborative-markdown-editor/internal/domain"
	"context"
	"time"

	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(userID uint64, document *domain.Document) error
	CreateUpdate(id uint64, userID uint64, content []byte) error
	ListDocumentByUserID(userID uint64, page, pageSize int) ([]domain.Document, DocumentsMeta, error)
	GetUserRole(docID uint64, userID uint64) (string, error)
	FindByID(id uint64) (*domain.Document, error)
	CurrentSeq(docID uint64, currentSeq *uint64) error
	CreateSnapshot(ctx context.Context, docID uint64, state []byte) error
	LastSnapshot(docID uint64, snapshot *domain.DocumentSnapshot) error
	LastSnapshotSeq(docID uint64, lastSnapshotSeq *uint64) error
	UpdatesFromSnapshot(docID uint64, snapshotSeq uint64, updates *[]domain.DocumentUpdate) error
	ListDocumentCollaborators(ctx context.Context, docID uint64) ([]collaboratorRow, error)
	AddCollaborator(ctx context.Context, docID uint64, userID uint64, role string) error
	UpdateCollaboratorRole(ctx context.Context, docID uint64, userID uint64, role string) error
	RemoveCollaborator(ctx context.Context, docID uint64, userID uint64) error
}

type DocumentRepositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new user repository
func NewRepository(db *gorm.DB) DocumentRepository {
	return &DocumentRepositoryImpl{db: db}
}

// Create creates a new user
func (r *DocumentRepositoryImpl) Create(userID uint64, document *domain.Document) error {
	document.UserID = userID
	document.CreatedAt = time.Now().UTC() // Use UTC for consistency
	document.UpdatedAt = time.Now().UTC()
	document.Collaborators = []domain.DocumentCollaborator{
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

func (r *DocumentRepositoryImpl) ListDocumentByUserID(userID uint64, page, pageSize int) ([]domain.Document, DocumentsMeta, error) {
	var documents []domain.Document
	var totalRecords int64

	// Count total records
	if err := r.db.Model(&domain.Document{}).Where("user_id = ?", userID).Count(&totalRecords).Error; err != nil {
		return documents, DocumentsMeta{}, err
	}

	offset := (page - 1) * pageSize
	err := r.db.Where("user_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Order("updated_at desc").
		Find(&documents).Error

	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))

	return documents, DocumentsMeta{
		Total:       totalRecords,
		PerPage:     pageSize,
		TotalPage:   totalPages,
		CurrentPage: page,
	}, err
}

func (r *DocumentRepositoryImpl) FindByID(id uint64) (*domain.Document, error) {
	var doc domain.Document
	err := r.db.First(&doc, id).Error
	return &doc, err
}

func (r *DocumentRepositoryImpl) GetUserRole(docID uint64, userID uint64) (string, error) {
	var role string
	err := r.db.Model(&domain.DocumentCollaborator{}).
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
		if err := tx.Create(&domain.DocumentUpdate{
			DocumentID:   id,
			Seq:          seq,
			UpdateBinary: content,
			UserID:       userID,
			CreatedAt:    now,
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
		if err := tx.Model(&domain.Document{}).
			Where("id = ?", docID).
			Select("update_seq").
			Scan(&lastSeq).Error; err != nil {
			return err
		}

		// check if already created
		var exists bool
		if err := tx.Model(&domain.DocumentSnapshot{}).
			Select("count(1) > 0").
			Where("document_id = ? AND seq = ?", docID, lastSeq).
			Find(&exists).Error; err != nil {
			return err
		}

		if exists {
			return nil // snapshot already exists
		}

		// create snapshot
		snapshot := domain.DocumentSnapshot{
			DocumentID:     docID,
			Seq:            lastSeq,
			SnapshotBinary: state,
		}

		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}

		// cleanup old updates
		if err := tx.Where("document_id = ? AND seq <= ?", docID, lastSeq).
			Delete(&domain.DocumentUpdate{}).Error; err != nil {
			return err
		}

		return nil
	})
	return err
}

func (r *DocumentRepositoryImpl) CurrentSeq(docID uint64, currentSeq *uint64) error {
	return r.db.Model(&domain.Document{}).
		Where("id = ?", docID).
		Select("update_seq").
		Scan(currentSeq).Error
}

func (r *DocumentRepositoryImpl) LastSnapshot(docID uint64, snapshot *domain.DocumentSnapshot) error {
	return r.db.Where("document_id = ?", docID).
		Order("seq DESC").
		First(&snapshot).Error
}

func (r *DocumentRepositoryImpl) LastSnapshotSeq(docID uint64, lastSnapshotSeq *uint64) error {
	return r.db.Model(&domain.DocumentSnapshot{}).
		Where("document_id = ?", docID).
		Select("COALESCE(MAX(seq), 0)").
		Scan(lastSnapshotSeq).Error
}

func (r *DocumentRepositoryImpl) UpdatesFromSnapshot(docID uint64, snapshotSeq uint64, updates *[]domain.DocumentUpdate) error {
	return r.db.Where("document_id = ? AND seq > ?", docID, snapshotSeq).
		Order("seq ASC").
		Limit(500).
		Find(&updates).Error
}

type collaboratorRow struct {
	UserID uint64
	Name   string
	Email  string
	Role   string
}

func (r *DocumentRepositoryImpl) ListDocumentCollaborators(ctx context.Context, docID uint64) ([]collaboratorRow, error) {
	var rows []collaboratorRow

	err := r.db.WithContext(ctx).
		Table("document_collaborators dc").
		Select(`
			u.id   AS user_id,
			u.name AS name,
			u.email AS email,
			dc.role AS role
		`).
		Joins("JOIN users u ON u.id = dc.user_id").
		Where("dc.document_id = ?", docID).
		Order("dc.added_at ASC").
		Scan(&rows).Error

	return rows, err
}

func (r *DocumentRepositoryImpl) AddCollaborator(ctx context.Context, docID uint64, userID uint64, role string) error {
	collab := domain.DocumentCollaborator{
		DocumentID: docID,
		UserID:     userID,
		Role:       role,
		AddedAt:    time.Now().UTC(),
	}

	return r.db.WithContext(ctx).Create(&collab).Error
}

func (r *DocumentRepositoryImpl) UpdateCollaboratorRole(ctx context.Context, docID uint64, userID uint64, role string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.DocumentCollaborator{}).
		Where("document_id = ? AND user_id = ?", docID, userID).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *DocumentRepositoryImpl) RemoveCollaborator(ctx context.Context, docID uint64, userID uint64) error {
	result := r.db.WithContext(ctx).
		Where("document_id = ? AND user_id = ?", docID, userID).
		Delete(&domain.DocumentCollaborator{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
