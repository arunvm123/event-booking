package main

import (
	"net/http"
	"time"

	"github.com/arunvm123/eventbooking/user-service/model"
	"github.com/arunvm123/eventbooking/user-service/repository"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	repo       repository.UserRepository
	jwtService *JWTService
}

func NewUserHandler(repo repository.UserRepository, jwtService *JWTService) *UserHandler {
	return &UserHandler{
		repo:       repo,
		jwtService: jwtService,
	}
}

// RegisterUser handles user registration
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
		return
	}

	// Create user in database
	user, err := h.repo.CreateUser(req.ToCreateUserRequest())
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{
				Error:   "validation_failed",
				Message: "Email already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
		})
		return
	}

	// Return user data (without password)
	response := model.RegisterResponse{
		UserResponse: user.ToUserResponse(),
	}

	c.JSON(http.StatusCreated, response)
}

// LoginUser handles user authentication
func (h *UserHandler) LoginUser(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
		return
	}

	// Get user by email
	user, err := h.repo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid email or password",
		})
		return
	}

	// Validate password
	if !h.repo.ValidatePassword(user, req.Password) {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error:   "authentication_failed",
			Message: "Invalid email or password",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
		return
	}

	// Return login response
	response := model.LoginResponse{
		AccessToken: token,
		ExpiresIn:   3600, // 1 hour in seconds
		User:        *user.ToUserResponse(),
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck handles health check endpoint
func (h *UserHandler) HealthCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := h.repo.GetDB().DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database ping failed",
		})
		return
	}

	response := model.HealthResponse{
		Status:    "healthy",
		Service:   "user-service",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
