package postgres

import (
	"errors"
	"log"

	"github.com/arunvm123/eventbooking/user-service/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(databaseURL string) (*PostgresUserRepository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the User model
	if err := db.AutoMigrate(&model.User{}); err != nil {
		return nil, err
	}

	log.Println("Database connected and User table migrated successfully")

	return &PostgresUserRepository{db: db}, nil
}

// CreateUser creates a new user with hashed password
func (r *PostgresUserRepository) CreateUser(req model.CreateUserRequest) (*model.User, error) {
	// Check if user already exists
	var existingUser model.User
	if err := r.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("email already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := model.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *PostgresUserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// ValidatePassword checks if the provided password matches the user's password
func (r *PostgresUserRepository) ValidatePassword(user *model.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// GetDB returns the database instance for health checks
func (r *PostgresUserRepository) GetDB() *gorm.DB {
	return r.db
}
