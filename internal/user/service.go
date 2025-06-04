package user

import (
	"collaborative-markdown-editor/internal/errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
	repository UserRepository
}

// NewService creates a new user service
func NewService(repository UserRepository) Service {
	return &DefaultService{repository: repository}
}

// Register registers a new user
func (s *DefaultService) Register(user *User) error {
	// Check if user with email already exists
	_, err := s.repository.FindByEmail(user.Email)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil {
		return errors.ErrUnprocessableEntity(nil).WithMessage("User already registered")
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.ErrUnprocessableEntity(err)
	}
	user.PasswordHash = string(hashedPassword)
	user.IsActive = true

	// Create user
	return s.repository.Create(user)
}

// Login authenticates a user
func (s *DefaultService) Login(email, password string) (*User, error) {
	// Find user by email
	user, err := s.repository.FindByEmail(email)
	if err != nil {
		return nil, errors.ErrUnauthorized(err).WithMessage("User not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.ErrUnauthorized(nil).WithMessage("User is not active")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.ErrUnprocessableEntity(err).WithMessage("Wrong password")
	}

	return user, nil
}

// GetUserByID gets a user by ID
func (s *DefaultService) GetUserByID(id uint) (*User, error) {
	return s.repository.FindByID(id)
}

// DeactivateUser deactivates a user
func (s *DefaultService) DeactivateUser(id uint) error {
	return s.repository.Deactivate(id)
}
