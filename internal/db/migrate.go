package db

import (
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/user"
	"context"
	"time"

	log "github.com/rs/zerolog/log"
)

// Migrate runs database migrations
func Migrate() {
	err := AppDb.AutoMigrate(
		&domain.User{},
		&domain.Document{},
		&domain.DocumentUpdate{},
		&domain.DocumentSnapshot{},
		&domain.DocumentVersion{},
		&domain.DocumentCollaborator{},
	)

	if err != nil {
		log.Fatal().Err(err).Msg("migration failed")
	}

	// db indexes
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_document_seq_unique ON document_updates (document_id, seq);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_document_snapshot_seq_unique ON document_snapshots (document_id, seq);`,
		`CREATE INDEX IF NOT EXISTS idx_updates_doc_created ON document_updates (document_id, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_versions_doc ON document_versions (document_id);`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_doc_seq ON document_snapshots (document_id, seq DESC);`,
	}
	err = RunSQL(statements)

	if err != nil {
		log.Fatal().Err(err).Msg("error running additional SQL statements")
	}
	log.Info().Msg("Database schema migrated successfully")
}

// SeedData seeds the database with initial data (for development only)
func SeedData() {
	// Create a test user if it doesn't exist
	userRepo := user.NewRepository(AppDb)

	testUser := &domain.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
		IsActive: true,
	}

	// 1 minute to finish seeding
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Check if user exists
	_, err := userRepo.FindByEmail(ctx, testUser.Email)
	if err != nil {
		userService := user.NewService(userRepo, nil)
		// User doesn't exist, create it
		if err := userService.Register(ctx, testUser); err != nil {
			log.Error().Err(err).Msg("Error creating test user")
		} else {
			log.Info().Str("email", testUser.Email).Msg("Created test user")
		}
	} else {
		log.Info().Str("email", testUser.Email).Msg("Test user already exists")
	}
}

func RunSQL(queries []string) error {
	for _, q := range queries {
		if err := AppDb.Exec(q).Error; err != nil {
			return err
		}
	}
	return nil
}
