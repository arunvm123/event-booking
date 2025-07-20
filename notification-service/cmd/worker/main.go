package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/arunvm123/eventbooking/notification-service/config"
	"github.com/arunvm123/eventbooking/notification-service/model"
	"github.com/segmentio/kafka-go"
)

var messagesProcessed int64

func main() {
	fmt.Println("Starting Notification Service Worker")

	// Load configuration
	cfg, err := config.Initialise("config.yaml", false)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Setup Kafka consumer
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.NotificationTopic,
		GroupID: cfg.Kafka.ConsumerGroup,
	})
	defer consumer.Close()

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

	// Start processing notifications
	fmt.Println("Notification processor worker started")
	if err := processNotifications(ctx, consumer); err != nil && err != context.Canceled {
		log.Fatal("Worker error:", err)
	}

	fmt.Println("Worker stopped gracefully")
}

func processNotifications(ctx context.Context, consumer *kafka.Reader) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read message from Kafka
			msg, err := consumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Process the notification
			if err := processNotification(msg); err != nil {
				log.Printf("Error processing notification: %v", err)
			}

			// Increment counter
			atomic.AddInt64(&messagesProcessed, 1)
		}
	}
}

func processNotification(msg kafka.Message) error {
	var notificationReq model.NotificationRequest
	if err := json.Unmarshal(msg.Value, &notificationReq); err != nil {
		return fmt.Errorf("failed to unmarshal notification request: %w", err)
	}

	log.Printf("Processing notification: %s for %s", notificationReq.Type, notificationReq.RecipientEmail)

	// Generate email based on notification type
	var emailTemplate *model.EmailTemplate
	switch notificationReq.Type {
	case "booking_confirmed":
		emailTemplate = notificationReq.GenerateBookingConfirmationEmail()
	case "booking_failed":
		emailTemplate = notificationReq.GenerateBookingFailedEmail()
	default:
		log.Printf("Unknown notification type: %s", notificationReq.Type)
		return nil
	}

	// Mock email sending (just log to console)
	if err := sendEmailMock(emailTemplate); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Successfully sent %s email to %s for booking %s",
		notificationReq.Type, notificationReq.RecipientEmail, notificationReq.BookingData.BookingID)

	return nil
}

// sendEmailMock simulates email sending by logging to console
func sendEmailMock(template *model.EmailTemplate) error {
	log.Printf("📧 MOCK EMAIL SENT:")
	log.Printf("   To: %s", template.To)
	log.Printf("   Subject: %s", template.Subject)
	log.Printf("   Body:\n%s", template.Body)

	return nil
}
