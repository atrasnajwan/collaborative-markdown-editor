package domain

import "time"

type Event struct {
	ID          string `gorm:"primaryKey"`
	ProcessedAt time.Time
}
