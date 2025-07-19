package repository

import (
	"github.com/arunvm123/eventbooking/user-service/model"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// CreateUser creates a new user with hashed password
	CreateUser(req model.CreateUserRequest) (*model.User, error)

	// GetUserByEmail retrieves a user by email
	GetUserByEmail(email string) (*model.User, error)

	// ValidatePassword checks if the provided password matches the user's password
	ValidatePassword(user *model.User, password string) bool

	// GetDB returns the database instance for health checks
	GetDB() *gorm.DB
}
