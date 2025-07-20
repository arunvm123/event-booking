package model

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// DATABASE ENTITIES (Internal - GORM only, no JSON tags)
// ============================================================================

// Booking represents the database model for bookings
type Booking struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index"`
	UserEmail     string    `gorm:"type:varchar(255);not null"`
	UserName      string    `gorm:"type:varchar(255);not null"`
	EventID       uuid.UUID `gorm:"type:uuid;not null;index"`
	EventName     string    `gorm:"type:varchar(255);not null"`
	Venue         string    `gorm:"type:varchar(255);not null"`
	EventDate     time.Time `gorm:"not null"`
	Seats         []string  `gorm:"type:text[];not null"`
	TotalAmount   float64   `gorm:"type:decimal(10,2);not null"`
	Status        string    `gorm:"type:varchar(20);not null;default:'processing'"`
	PaymentStatus string    `gorm:"type:varchar(20);not null;default:'pending'"`
	HoldID        uuid.UUID `gorm:"type:uuid;not null;index"`
	ErrorMessage  *string   `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	ConfirmedAt   *time.Time
	FailedAt      *time.Time
}

// TableName sets the table name for GORM
func (Booking) TableName() string {
	return "bookings"
}

// ============================================================================
// REPOSITORY DATA TRANSFER OBJECTS (Internal - no JSON tags)
// ============================================================================

// CreateBookingRequest represents the data needed to create a booking
type CreateBookingRequest struct {
	UserID        uuid.UUID
	UserEmail     string
	UserName      string
	EventID       uuid.UUID
	EventName     string
	Venue         string
	EventDate     time.Time
	Seats         []string
	TotalAmount   float64
	HoldID        uuid.UUID
	PaymentMethod string
}

// UpdateBookingStatusRequest represents a booking status update
type UpdateBookingStatusRequest struct {
	BookingID     uuid.UUID
	Status        string
	PaymentStatus string
	ErrorMessage  *string
	ConfirmedAt   *time.Time
	FailedAt      *time.Time
}

// BookingFilter represents filtering options for booking queries
type BookingFilter struct {
	UserID uuid.UUID
	Status string
	Limit  int
	Offset int
}

// ============================================================================
// API DATA TRANSFER OBJECTS (External - JSON tags for HTTP)
// ============================================================================

// SubmitBookingRequest represents the API request to submit a booking
type SubmitBookingRequest struct {
	HoldID      uuid.UUID   `json:"hold_id" binding:"required"`
	PaymentInfo PaymentInfo `json:"payment_info" binding:"required"`
}

// PaymentInfo represents payment information in booking request
type PaymentInfo struct {
	PaymentMethod string  `json:"payment_method" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

// BookingResponse represents the API response after booking submission
type BookingResponse struct {
	BookingID     uuid.UUID `json:"booking_id"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	EstimatedTime string    `json:"estimated_time"`
	StatusURL     string    `json:"status_url"`
	StreamURL     string    `json:"stream_url"`
}

// BookingStatusResponse represents the detailed booking status response
type BookingStatusResponse struct {
	BookingID     uuid.UUID            `json:"booking_id"`
	Status        string               `json:"status"`
	Event         *BookingEventDetails `json:"event,omitempty"`
	Seats         []string             `json:"seats,omitempty"`
	TotalAmount   float64              `json:"total_amount,omitempty"`
	PaymentStatus string               `json:"payment_status,omitempty"`
	ErrorMessage  *string              `json:"error_message,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	ConfirmedAt   *time.Time           `json:"confirmed_at,omitempty"`
	FailedAt      *time.Time           `json:"failed_at,omitempty"`
}

// BookingEventDetails represents event information in booking status
type BookingEventDetails struct {
	EventID   uuid.UUID `json:"event_id"`
	Name      string    `json:"name"`
	Venue     string    `json:"venue"`
	EventDate time.Time `json:"event_date"`
}

// UserBookingsResponse represents the list of user bookings
type UserBookingsResponse struct {
	Bookings []UserBookingSummary `json:"bookings"`
	Total    int                  `json:"total"`
}

// UserBookingSummary represents a summary of user booking for listing
type UserBookingSummary struct {
	BookingID   uuid.UUID `json:"booking_id"`
	Status      string    `json:"status"`
	EventName   string    `json:"event_name"`
	Venue       string    `json:"venue"`
	EventDate   time.Time `json:"event_date"`
	Seats       []string  `json:"seats"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

// BookingStatusUpdate represents real-time status updates for SSE
type BookingStatusUpdate struct {
	BookingID uuid.UUID `json:"booking_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ============================================================================
// KAFKA MESSAGE STRUCTURES
// ============================================================================

// BookingRequest represents the message sent to Kafka booking topic
type BookingRequest struct {
	BookingID   uuid.UUID   `json:"booking_id"`
	UserID      uuid.UUID   `json:"user_id"`
	UserEmail   string      `json:"user_email"`
	UserName    string      `json:"user_name"`
	HoldID      uuid.UUID   `json:"hold_id"`
	EventID     uuid.UUID   `json:"event_id"`
	EventName   string      `json:"event_name"`
	Venue       string      `json:"venue"`
	EventDate   time.Time   `json:"event_date"`
	Seats       []string    `json:"seats"`
	PaymentInfo PaymentInfo `json:"payment_info"`
	Timestamp   time.Time   `json:"timestamp"`
}

// NotificationRequest represents the message sent to notification topic
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
// CONVERSION METHODS
// ============================================================================

// ToBookingStatusResponse converts a Booking entity to a status response
func (b *Booking) ToBookingStatusResponse() *BookingStatusResponse {
	response := &BookingStatusResponse{
		BookingID:     b.ID,
		Status:        b.Status,
		PaymentStatus: b.PaymentStatus,
		CreatedAt:     b.CreatedAt,
		ConfirmedAt:   b.ConfirmedAt,
		FailedAt:      b.FailedAt,
		ErrorMessage:  b.ErrorMessage,
	}

	if b.Status == "confirmed" || b.Status == "processing" {
		response.Event = &BookingEventDetails{
			EventID:   b.EventID,
			Name:      b.EventName,
			Venue:     b.Venue,
			EventDate: b.EventDate,
		}
		response.Seats = b.Seats
		response.TotalAmount = b.TotalAmount
	}

	return response
}

// ToUserBookingSummary converts a Booking entity to a user booking summary
func (b *Booking) ToUserBookingSummary() UserBookingSummary {
	return UserBookingSummary{
		BookingID:   b.ID,
		Status:      b.Status,
		EventName:   b.EventName,
		Venue:       b.Venue,
		EventDate:   b.EventDate,
		Seats:       b.Seats,
		TotalAmount: b.TotalAmount,
		CreatedAt:   b.CreatedAt,
	}
}
