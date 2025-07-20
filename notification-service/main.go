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
	// Load configuration
	cfg, err := config.Initialise("config.yaml", false)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
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
