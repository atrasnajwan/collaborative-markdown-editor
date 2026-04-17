package event

import (
	"collaborative-markdown-editor/internal/domain"
	"context"

	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) EventRepository {
	return EventRepository{db: db}
}

func (r EventRepository) Create(ctx context.Context, eventID string) error {
	processed := domain.Event{
		ID: eventID,
	}

	return r.db.WithContext(ctx).Create(processed).Error
}
