package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/arunvm123/eventbooking/booking-service/cache"
	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/arunvm123/eventbooking/booking-service/repository"
	"github.com/arunvm123/eventbooking/booking-service/service"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type BookingProcessor struct {
	repo         repository.BookingRepository
	cache        cache.CacheRepository
	eventService service.EventService
	kafkaWriter  *kafka.Writer
	consumer     *kafka.Reader
}

func NewBookingProcessor(
	repo repository.BookingRepository,
	cache cache.CacheRepository,
	eventService service.EventService,
	kafkaWriter *kafka.Writer,
	consumer *kafka.Reader,
) *BookingProcessor {
	return &BookingProcessor{
		repo:         repo,
		cache:        cache,
		eventService: eventService,
		kafkaWriter:  kafkaWriter,
		consumer:     consumer,
	}
}

// Start begins processing booking requests from Kafka
func (p *BookingProcessor) Start(ctx context.Context) error {
	log.Println("Starting booking processor...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Booking processor shutting down...")
			return ctx.Err()
		default:
			// Read message from Kafka
			msg, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Process the booking
			if err := p.processBooking(msg); err != nil {
				log.Printf("Error processing booking: %v", err)
				// Could implement retry logic here
			}
		}
	}
}

// processBooking handles individual booking requests
func (p *BookingProcessor) processBooking(msg kafka.Message) error {
	var bookingReq model.BookingRequest
	if err := json.Unmarshal(msg.Value, &bookingReq); err != nil {
		return fmt.Errorf("failed to unmarshal booking request: %w", err)
	}

	log.Printf("Processing booking: %s for user: %s", bookingReq.BookingID, bookingReq.UserID)

	// Update status to processing
	p.updateBookingStatus(bookingReq.BookingID, "processing", "payment", "Processing payment...", nil, nil)

	// Step 1: Simulate payment processing
	if err := p.processPayment(bookingReq); err != nil {
		// Payment failed - release hold and mark booking as failed
		p.eventService.ReleaseHold(bookingReq.HoldID)
		failTime := time.Now()
		errMsg := fmt.Sprintf("Payment failed: %s", err.Error())
		p.updateBookingStatus(bookingReq.BookingID, "failed", "failed", errMsg, nil, &failTime)
		p.sendNotification(bookingReq, "booking_failed", errMsg)
		return err
	}

	// Step 2: Confirm hold with Event Service (mark seats as booked)
	if err := p.eventService.ConfirmHold(bookingReq.HoldID); err != nil {
		// Hold confirmation failed - could be expired, seats taken, etc.
		failTime := time.Now()
		errMsg := fmt.Sprintf("Failed to confirm seats: %s", err.Error())
		p.updateBookingStatus(bookingReq.BookingID, "failed", "refund_pending", errMsg, nil, &failTime)
		p.sendNotification(bookingReq, "booking_failed", errMsg)
		return err
	}

	// Step 3: Mark booking as confirmed
	confirmTime := time.Now()
	p.updateBookingStatus(bookingReq.BookingID, "confirmed", "completed", "Booking confirmed successfully", &confirmTime, nil)

	// Step 4: Send confirmation notification
	p.sendNotification(bookingReq, "booking_confirmed", "Your booking has been confirmed!")

	log.Printf("Successfully processed booking: %s", bookingReq.BookingID)
	return nil
}

// processPayment simulates payment processing
func (p *BookingProcessor) processPayment(bookingReq model.BookingRequest) error {
	// Simulate payment processing time
	time.Sleep(2 * time.Second)

	// Simulate payment validation
	if bookingReq.PaymentInfo.Amount <= 0 {
		return fmt.Errorf("invalid payment amount: %f", bookingReq.PaymentInfo.Amount)
	}

	if bookingReq.PaymentInfo.PaymentMethod == "" {
		return fmt.Errorf("payment method is required")
	}

	// Simulate payment failure rate (5% failure for demo)
	// In real implementation, this would call actual payment gateway
	if time.Now().UnixNano()%20 == 0 {
		return fmt.Errorf("payment gateway declined transaction")
	}

	log.Printf("Payment processed successfully for booking: %s, amount: $%.2f",
		bookingReq.BookingID, bookingReq.PaymentInfo.Amount)
	return nil
}

// updateBookingStatus updates booking status in both database and cache
func (p *BookingProcessor) updateBookingStatus(bookingID uuid.UUID, status, paymentStatus, message string, confirmedAt, failedAt *time.Time) {
	// Update database
	updateReq := model.UpdateBookingStatusRequest{
		BookingID:     bookingID,
		Status:        status,
		PaymentStatus: paymentStatus,
		ConfirmedAt:   confirmedAt,
		FailedAt:      failedAt,
	}

	if status == "failed" {
		updateReq.ErrorMessage = &message
	}

	if err := p.repo.UpdateBookingStatus(updateReq); err != nil {
		log.Printf("Failed to update booking status in database: %v", err)
	}

	// Update cache for SSE
	statusUpdate := &model.BookingStatusUpdate{
		BookingID: bookingID,
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now(),
	}

	if err := p.cache.SetBookingStatus(bookingID, statusUpdate, 24*time.Hour); err != nil {
		log.Printf("Failed to update booking status in cache: %v", err)
	}
}

// sendNotification sends notification to Kafka notification topic
func (p *BookingProcessor) sendNotification(bookingReq model.BookingRequest, notificationType, message string) {
	notification := model.NotificationRequest{
		Type:           notificationType,
		RecipientEmail: bookingReq.UserEmail,
		BookingData: model.NotificationBookingData{
			BookingID:   bookingReq.BookingID,
			EventName:   bookingReq.EventName,
			Venue:       bookingReq.Venue,
			EventDate:   bookingReq.EventDate,
			Seats:       bookingReq.Seats,
			TotalAmount: bookingReq.PaymentInfo.Amount,
			UserName:    bookingReq.UserName,
		},
		Timestamp: time.Now(),
	}

	notificationBytes, _ := json.Marshal(notification)
	p.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(bookingReq.BookingID.String()),
			Value: notificationBytes,
		})
}
