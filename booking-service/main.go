package main

import (
	"fmt"
	"log"

	"github.com/arunvm123/eventbooking/booking-service/config"
)

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

	// Setup router with all dependencies
	router := SetupRouter(cfg)

	// Start server
	fmt.Printf("Starting Booking Service API on port %s\n", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
