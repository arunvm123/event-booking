package main

import (
	"log"

	"github.com/arunvm123/eventbooking/user-service/config"
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

	r := SetupRouter(cfg)
	log.Printf("User Service running on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
