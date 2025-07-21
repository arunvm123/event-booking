package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/arunvm123/eventbooking/notification-service/config"
	"github.com/arunvm123/eventbooking/notification-service/model"
	"github.com/gin-gonic/gin"
)

var messagesProcessed int64

func main() {
	// Initialize configuration
	// Try to load from config.yaml first, fallback to environment variables
	cfg, err := config.Initialise("config.yaml", false)
	if err != nil {
		// If config file fails, try environment variables
		log.Printf("Config file not found or invalid, using environment variables: %v", err)
		cfg, err = config.Initialise("", true)
		if err != nil {
			log.Fatal("Failed to load configuration:", err)
		}
	}

	// Setup Gin router
	r := gin.Default()

	// Health check endpoint only
	r.GET("/health", func(c *gin.Context) {
		response := model.HealthResponse{
			Status:            "healthy",
			Service:           "notification-service",
			Timestamp:         time.Now(),
			MessagesProcessed: atomic.LoadInt64(&messagesProcessed),
		}
		c.JSON(http.StatusOK, response)
	})

	// Start server
	fmt.Printf("Starting Notification Service API on port %s\n", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
