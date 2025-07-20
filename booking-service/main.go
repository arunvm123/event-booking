package main

import (
	"fmt"
	"log"

	"github.com/arunvm123/eventbooking/booking-service/config"
)

func main() {
	// Load configuration
	cfg, err := config.Initialise("config.yaml", false)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Setup router with all dependencies
	router := SetupRouter(cfg)

	// Start server
	fmt.Printf("Starting Booking Service API on port %s\n", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
