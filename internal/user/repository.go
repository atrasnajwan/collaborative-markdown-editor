package user

import "gorm.io/gorm"

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id uint64) (*User, error)
	Deactivate(id uint64) error
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
func (r *UserRepositoryImpl) Create(user *User) error {
	return r.db.Create(user).Error
}

// FindByEmail finds a user by email
func (r *UserRepositoryImpl) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, err
}

// FindByID finds a user by ID
func (r *UserRepositoryImpl) FindByID(id uint64) (*User, error) {
	var user User
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