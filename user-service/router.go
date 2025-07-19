package main

import (
	"log"

	"github.com/arunvm123/eventbooking/user-service/config"
	"github.com/arunvm123/eventbooking/user-service/repository/postgres"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	// Initialize repository
	repo, err := postgres.NewUserRepository(cfg.Database.GetDatabaseURL())
	if err != nil {
		log.Fatal("Failed to initialize repository:", err)
	}

	// Initialize JWT service
	jwtService := NewJWTService(cfg.JWTSecret)

	// Initialize handlers
	userHandler := NewUserHandler(repo, jwtService)

	// Setup Gin router
	r := gin.Default()

	// Add middleware
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())

	// Health check endpoint (no auth required)
	r.GET("/health", userHandler.HealthCheck)

	// API routes
	api := r.Group("/api")
	users := api.Group("/users")

	// Public endpoints (no auth required)
	users.POST("/register", userHandler.RegisterUser)
	users.POST("/login", userHandler.LoginUser)

	return r
}
