package user

import (
	"collaborative-markdown-editor/internal/errors"

	"golang.org/x/crypto/bcrypt"
)

// Service defines the interface for user business logic
type Service interface {
	Register(user *User) error
	Login(email, password string) (*User, error)
	GetUserByID(id uint) (*User, error)
	DeactivateUser(id uint) error
}

// DefaultService implements Service
type DefaultService struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &DefaultService{repo: repo}
}

// Register registers a new user
func (s *DefaultService) Register(user *User) error {
	// Check if user with email already exists
	_, err := s.repo.FindByEmail(user.Email)
	if err == nil {
		return errors.ErrUnprocessableEntity(nil)
	}

	// Create user
	return s.repo.Create(user)
}

// Login authenticates a user
func (s *DefaultService) Login(email, password string) (*User, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.ErrUnauthorized(err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.ErrUnauthorized(nil)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.ErrUnauthorized(err)
	}

	return user, nil
}

// GetUserByID gets a user by ID
func (s *DefaultService) GetUserByID(id uint) (*User, error) {
	return s.repo.FindByID(id)
}

// DeactivateUser deactivates a user
func (s *DefaultService) DeactivateUser(id uint) error {
	return s.repo.Deactivate(id)
}
