package db

import (
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/user"
	"log"
)

// Migrate runs database migrations
func Migrate() {
	err := AppDb.AutoMigrate(
		&user.User{},
		&document.Document{},
		&document.DocumentEdit{},
		&document.DocumentPermission{},
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Database schema migrated successfully")
}

// SeedData seeds the database with initial data (for development only)
func SeedData() {
	// Create a test user if it doesn't exist
	userRepo := user.NewRepository(AppDb)
	
	testUser := &user.User{
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
