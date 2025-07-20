package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// KAFKA MESSAGE STRUCTURES (From Booking Service)
// ============================================================================

// NotificationRequest represents the message consumed from Kafka notification topic
type NotificationRequest struct {
	Type           string                  `json:"type"`
	RecipientEmail string                  `json:"recipient_email"`
	BookingData    NotificationBookingData `json:"booking_data"`
	Timestamp      time.Time               `json:"timestamp"`
}

// NotificationBookingData represents booking data for notifications
type NotificationBookingData struct {
	BookingID   uuid.UUID `json:"booking_id"`
	EventName   string    `json:"event_name"`
	Venue       string    `json:"venue"`
	EventDate   time.Time `json:"event_date"`
	Seats       []string  `json:"seats"`
	TotalAmount float64   `json:"total_amount"`
	UserName    string    `json:"user_name"`
}

// ============================================================================
// EMAIL TEMPLATES
// ============================================================================

// EmailTemplate represents an email to be sent (logged to console)
type EmailTemplate struct {
	To      string
	Subject string
	Body    string
}

// ============================================================================
// EMAIL GENERATION METHODS
// ============================================================================

// GenerateBookingConfirmationEmail creates simple email content for booking confirmation
func (nr *NotificationRequest) GenerateBookingConfirmationEmail() *EmailTemplate {
	subject := "Booking Confirmed - " + nr.BookingData.EventName

	body := "Dear " + nr.BookingData.UserName + ",\n\n" +
		"Your booking has been confirmed!\n\n" +
		"Event: " + nr.BookingData.EventName + "\n" +
		"Venue: " + nr.BookingData.Venue + "\n" +
		"Date: " + nr.BookingData.EventDate.Format("2006-01-02 15:04") + "\n" +
		"Seats: " + fmt.Sprintf("%v", nr.BookingData.Seats) + "\n" +
		"Amount: $" + fmt.Sprintf("%.2f", nr.BookingData.TotalAmount) + "\n" +
		"Booking ID: " + nr.BookingData.BookingID.String() + "\n\n" +
		"Thank you for your booking!\n\n" +
		"Event Booking System"

	return &EmailTemplate{
		To:      nr.RecipientEmail,
		Subject: subject,
		Body:    body,
	}
}

// GenerateBookingFailedEmail creates simple email content for booking failure
func (nr *NotificationRequest) GenerateBookingFailedEmail() *EmailTemplate {
	subject := "Booking Failed - " + nr.BookingData.EventName

	body := "Dear " + nr.BookingData.UserName + ",\n\n" +
		"We're sorry, but your booking could not be completed.\n\n" +
		"Event: " + nr.BookingData.EventName + "\n" +
		"Booking ID: " + nr.BookingData.BookingID.String() + "\n\n" +
		"Any charges will be refunded within 3-5 business days.\n" +
		"Please try booking again or contact support.\n\n" +
		"Event Booking System"

	return &EmailTemplate{
		To:      nr.RecipientEmail,
		Subject: subject,
		Body:    body,
	}
}

// ============================================================================
// API DATA TRANSFER OBJECTS (External - JSON tags for HTTP)
// ============================================================================

// HealthResponse represents the health check response
type HealthResponse struct {
	Status            string    `json:"status"`
	Service           string    `json:"service"`
	Timestamp         time.Time `json:"timestamp"`
	MessagesProcessed int64     `json:"messages_processed"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
