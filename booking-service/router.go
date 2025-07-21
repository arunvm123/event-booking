package main

import (
	"log"

	"github.com/arunvm123/eventbooking/booking-service/cache/redis"
	"github.com/arunvm123/eventbooking/booking-service/config"
	"github.com/arunvm123/eventbooking/booking-service/repository/postgres"
	httpservice "github.com/arunvm123/eventbooking/booking-service/service/http"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	// Initialize repository
	repo, err := postgres.NewBookingRepository(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize repository:", err)
	}

	// Initialize cache
	cache, err := redis.NewRedisCacheRepository(cfg.Redis.GetRedisURL(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal("Failed to initialize cache:", err)
	}

	// Initialize Event Service client with connection pooling
	eventService := httpservice.NewHTTPEventServiceWithConfig(&cfg.EventService, cfg.JWTSecret)

	// Initialize Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.BookingTopic,
		Balancer: &kafka.LeastBytes{},
	}

	// Initialize JWT service
	jwtService := NewJWTService(cfg.JWTSecret)

	// Initialize handlers
	bookingHandler := NewBookingHandler(repo, cache, kafkaWriter, eventService)

	// Setup Gin router
	r := gin.Default()

	// Add middleware
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())

	// Health check endpoint (no auth required)
	r.GET("/health", bookingHandler.HealthCheck)

	// API routes
	api := r.Group("/api")

	// Protected endpoints (require authentication)
	protected := api.Group("")
	protected.Use(AuthMiddleware(jwtService))

	// Booking endpoints
	protected.POST("/booking", bookingHandler.SubmitBooking)
	protected.GET("/booking/:bookingId/status", bookingHandler.GetBookingStatus)
	protected.GET("/booking/:bookingId/stream", bookingHandler.StreamBookingStatus)
	protected.GET("/bookings", bookingHandler.ListUserBookings)

	return r
}
