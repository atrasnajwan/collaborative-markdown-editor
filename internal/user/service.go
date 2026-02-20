package user

import (
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/redis"
	"context"
	"fmt"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service defines the interface for user business logic
type Service interface {
	Register(ctx context.Context, user *domain.User) error
	UpdateUser(ctx context.Context, userID uint64, req UpdateProfileRequest) (domain.SafeUser, error)
	ChangePassword(ctx context.Context, userID uint64, req ChangePasswordRequest) error
	Login(ctx context.Context, email, password string) (*domain.User, error)
	Logout(ctx context.Context, userID uint64)
	GetUserByID(ctx context.Context, id uint64) (*domain.User, error)
	DeactivateUser(ctx context.Context,id uint64) error
	SearchUsers(ctx context.Context, query string) ([]domain.SafeUser, error)
}

// DefaultService implements Service
type DefaultService struct {
	repository UserRepository
	cache *redis.Cache
}

// NewService creates a new user service
func NewService(repository UserRepository, cache *redis.Cache) Service {
	return &DefaultService{repository: repository, cache: cache}
}

// Register registers a new user
func (s *DefaultService) Register(ctx context.Context, user *domain.User) error {
	// Check if user with email already exists
	_, err := s.repository.FindByEmail(ctx, user.Email)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil {
		return errors.UnprocessableEntity("User already registered", nil)
	}

	// Hash the password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedPassword)
	user.IsActive = true

	// Create user
	return s.repository.Create(ctx, user)
}

func (s *DefaultService) UpdateUser(ctx context.Context, userID uint64, req UpdateProfileRequest) (domain.SafeUser, error) {
    updateData := make(map[string]interface{})

    if req.Name != nil {
        updateData["name"] = *req.Name
    }
    
    if req.Email != nil {
        updateData["email"] = *req.Email
    }

    user, err := s.repository.UpdateFields(ctx, userID, updateData)
    if err != nil {
        // Handle unique constraint for email
        if strings.Contains(err.Error(), "duplicate key") {
            return domain.SafeUser{}, errors.Conflict("Email already in use", err)
        }
        return domain.SafeUser{}, err
    }

    return user.ToSafeUser(), nil
}

func (s *DefaultService) ChangePassword(ctx context.Context, userID uint64, req ChangePasswordRequest) error {
    user, _ := s.repository.FindByID(ctx, userID)
    
    // Verify old password
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
        return errors.Unauthorized("Current password incorrect", nil)
    }

    // Hash new password
    hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
    if err != nil {
		return err
	}

    // Update and increment TokenVersion (to log out other devices)
    _, err = s.repository.UpdateFields(ctx, userID, map[string]interface{}{
        "password_hash": string(hashed),
        "token_version": gorm.Expr("token_version + 1"),
    })
	return err
}

// Login authenticates a user
func (s *DefaultService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	// Find user by email
	user, err := s.repository.FindByEmail(ctx, email)
	if err != nil {
		return nil, errors.Unauthorized("User not found!", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil,errors.Unauthorized("User not active!", err)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.UnprocessableEntity("Wrong Password!", err)
	}
	return user, nil
}

func (s *DefaultService) Logout(ctx context.Context, userID uint64) {
	err := s.repository.UpdateTokenVersion(ctx, userID)
	if err != nil {
		log.Printf("%v\n", err.Error())
	}
	// invalidate cache/redis
	cacheKey := fmt.Sprintf("user:version:%d", userID)
	s.cache.Invalidate(ctx, cacheKey)
}

// GetUserByID gets a user by ID
func (s *DefaultService) GetUserByID(ctx context.Context, id uint64) (*domain.User, error) {
	return s.repository.FindByID(ctx, id)
}

// DeactivateUser deactivates a user
func (s *DefaultService) DeactivateUser(ctx context.Context, id uint64) error {
	return s.repository.Deactivate(ctx, id)
}

func (s *DefaultService) SearchUsers(ctx context.Context, query string) ([]domain.SafeUser, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return []domain.SafeUser{}, nil
	}

	users, err := s.repository.SearchUsers(ctx, query, 10)
	if err != nil {
		return nil, err
	}

	result := make([]domain.SafeUser, 0, len(users))
	for _, u := range users {
		result = append(result, u.ToSafeUser())
	}

	return result, nil
}
