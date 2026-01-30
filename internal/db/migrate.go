package db

import (
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/user"
	"log"
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
		log.Fatal(err)	
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
        log.Fatal(err)
    }
	log.Println("Database schema migrated successfully")
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

	// Check if user exists
	_, err := userRepo.FindByEmail(testUser.Email)
	if err != nil {
		userService := user.NewService(userRepo)
		// User doesn't exist, create it
		if err := userService.Register(testUser); err != nil {
			log.Printf("Error creating test user: %v", err)
		} else {
			log.Printf("Created test user: %s", testUser.Email)
		}
	} else {
		log.Printf("Test user already exists: %s", testUser.Email)
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