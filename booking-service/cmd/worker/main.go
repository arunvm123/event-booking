package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arunvm123/eventbooking/booking-service/cache/redis"
	"github.com/arunvm123/eventbooking/booking-service/config"
	"github.com/arunvm123/eventbooking/booking-service/repository/postgres"
	"github.com/arunvm123/eventbooking/booking-service/service/http"
	"github.com/arunvm123/eventbooking/booking-service/worker"
	"github.com/segmentio/kafka-go"
)

func main() {
	fmt.Println("Starting Booking Service Worker")

	// Load configuration (fallback to env variables if config file not found)
	cfg, err := config.Initialise("config.yaml", false)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

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

	// Initialize Event Service client
	eventService := http.NewHTTPEventService(cfg.EventService.BaseURL, cfg.JWTSecret)

	// Initialize Kafka writer for notifications
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers...),
		Topic:    cfg.Kafka.NotificationTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// Setup Kafka consumer
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.BookingTopic,
		GroupID: cfg.Kafka.ConsumerGroup,
	})
	defer consumer.Close()

	// Create booking processor
	processor := worker.NewBookingProcessor(repo, cache, eventService, kafkaWriter, consumer)

	// Graceful shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("Received shutdown signal, stopping worker...")
		cancel()
	}()

	// Start worker
	fmt.Println("Booking processor worker started")
	if err := processor.Start(ctx); err != nil && err != context.Canceled {
		log.Fatal("Worker error:", err)
	}

	fmt.Println("Worker stopped gracefully")
}
