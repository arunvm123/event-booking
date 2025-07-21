package model

import (
	"time"
)

// ===============================
// Database Entities (Internal)
// ===============================

// User represents the user entity in the database
type User struct {
	ID           string `gorm:"primary_key;default:gen_random_uuid()"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	FirstName    string `gorm:"not null"`
	LastName     string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ToUserResponse converts database User to API response
func (u *User) ToUserResponse() *UserResponse {
	return &UserResponse{
		UserID:    u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		CreatedAt: u.CreatedAt,
	}
}

// ===============================
// Repository DTOs (Internal)
// ===============================

// CreateUserRequest represents input for creating a user in repository layer
type CreateUserRequest struct {
	ID        string
	Email     string
	Password  string // Plain text password (will be hashed in repository)
	FirstName string
	LastName  string
}

// ===============================
// API DTOs (External)
// ===============================

// RegisterRequest represents the user registration request from API
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// ToCreateUserRequest converts API request to repository request
func (r *RegisterRequest) ToCreateUserRequest() CreateUserRequest {
	return CreateUserRequest{
		Email:     r.Email,
		Password:  r.Password,
		FirstName: r.FirstName,
		LastName:  r.LastName,
	}
}

// LoginRequest represents the user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse represents user data in API responses
type UserResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterResponse represents the response for user registration
type RegisterResponse struct {
	*UserResponse
}

// LoginResponse represents the response for user login
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        UserResponse `json:"user"`
}

// ErrorResponse represents error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}
