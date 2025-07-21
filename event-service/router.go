package main

import (
	"log"

	"github.com/arunvm123/eventbooking/event-service/cache/redis"
	"github.com/arunvm123/eventbooking/event-service/config"
	"github.com/arunvm123/eventbooking/event-service/repository/postgres"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	// Initialize repository
	repo, err := postgres.NewEventRepository(cfg.Database.GetDatabaseURL())
	if err != nil {
		log.Fatal("Failed to initialize repository:", err)
	}

	// Initialize cache
	cache, err := redis.NewRedisCacheRepository(cfg.Redis.GetRedisURL(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal("Failed to initialize cache:", err)
	}

	// Initialize JWT service
	jwtService := NewJWTService(cfg.JWTSecret)

	// Initialize handlers
	eventHandler := NewEventHandler(repo, cache)

	// Setup Gin router
	r := gin.Default()

	// Add middleware
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())

	// Health check endpoint (no auth required)
	r.GET("/health", eventHandler.HealthCheck)

	// API routes
	api := r.Group("/api")
	events := api.Group("/events")

	// Public endpoints (no auth required)
	events.GET("", eventHandler.ListEvents)
	events.GET("/:id", eventHandler.GetEvent)

	// Protected endpoints (require authentication)
	protected := events.Group("")
	protected.Use(AuthMiddleware(jwtService))

	// Event management (authenticated users only)
	protected.POST("", eventHandler.CreateEvent)

	// Seat operations (authenticated users only)
	protected.POST("/:id/hold", eventHandler.HoldSeats)
	protected.GET("/holds/:holdId", eventHandler.GetHoldDetails)
	protected.DELETE("/holds/:holdId", eventHandler.ReleaseHold)
	protected.POST("/holds/:holdId/confirm", eventHandler.ConfirmHold)

	return r
}
