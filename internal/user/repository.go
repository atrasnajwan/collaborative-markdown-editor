package user

import (
	"collaborative-markdown-editor/internal/domain"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(user *domain.User) error
	UpdateFields(ctx context.Context, userID uint64, updates map[string]interface{}) (*domain.User, error)
	FindByEmail(email string) (*domain.User, error)
	FindByID(id uint64) (*domain.User, error)
	Deactivate(id uint64) error
	UpdateTokenVersion(id uint64) error 
	SearchUsers(ctx context.Context, query string, limit int) ([]domain.User, error) 
}

// UserRepositoryImpl implements User
type UserRepositoryImpl struct{
	db *gorm.DB
}

// NewRepository creates a new user repository
func NewRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

// Create creates a new user
func (r *UserRepositoryImpl) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepositoryImpl) UpdateFields(ctx context.Context, userID uint64, updates map[string]interface{}) (*domain.User, error) {
    var user domain.User
    
	result := r.db.WithContext(ctx).
        Model(&user).
        Clauses(clause.Returning{}). // tells Postgres to return the updated row
        Where("id = ?", userID).
        Updates(updates)

    if result.Error != nil {
        return nil, result.Error
    }
    if result.RowsAffected == 0 {
        return nil, gorm.ErrRecordNotFound
    }

    return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepositoryImpl) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

// FindByID finds a user by ID
func (r *UserRepositoryImpl) FindByID(id uint64) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	return &user, err
}

// Deactivate deactivates a user
func (r *UserRepositoryImpl) Deactivate(id uint64) error {
	user, err := r.FindByID(id)
	if err != nil {
		return err
	}

	user.IsActive = false
	return r.db.Save(user).Error
}

func (r *UserRepositoryImpl) UpdateTokenVersion(id uint64) error {
	return r.db.Model(&domain.User{}).
		Where("id = ?", id).
		Update("token_version", gorm.Expr("token_version + 1")).Error
}

func (r *UserRepositoryImpl) SearchUsers(ctx context.Context, query string, limit int) ([]domain.User, error) {
	var users []domain.User

	q := "%" + strings.ToLower(query) + "%"

	err := r.db.WithContext(ctx).
		Where(
			"LOWER(name) LIKE ? OR LOWER(email) LIKE ?",
			q, q,
		).
		Order("name ASC").
		Limit(limit).
		Find(&users).Error

	return users, err
}
