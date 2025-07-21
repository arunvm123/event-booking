package model

import (
	"time"

	"github.com/lib/pq"
)

// ===============================
// Database Entities (Internal)
// ===============================

// Event represents the event entity in the database
type Event struct {
	ID           string `gorm:"type:text;primary_key"`
	Name         string `gorm:"not null"`
	Description  string
	Venue        string    `gorm:"not null"`
	City         string    `gorm:"not null"`
	Category     string    `gorm:"not null"`
	EventDate    time.Time `gorm:"not null"`
	TotalSeats   int       `gorm:"not null"`
	PricePerSeat float64   `gorm:"not null"`
	CreatedBy    string    `gorm:"type:text;not null"` // User ID from User Service
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Seat represents the seat entity in the database
type Seat struct {
	ID         string  `gorm:"type:text;primary_key"`
	EventID    string  `gorm:"type:text;not null"`
	SeatNumber string  `gorm:"not null"`
	Status     string  `gorm:"default:'available'"` // available, held, booked
	HoldID     *string `gorm:"type:text"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Event Event `gorm:"foreignKey:EventID"`
}

// Hold represents the hold entity in the database
type Hold struct {
	ID          string         `gorm:"type:text;primary_key"`
	UserID      string         `gorm:"type:text;not null"` // User ID from User Service
	EventID     string         `gorm:"type:text;not null"`
	SeatNumbers pq.StringArray `gorm:"type:text[]"`
	ExpiresAt   time.Time      `gorm:"not null"`
	Status      string         `gorm:"default:'active'"` // active, confirmed, expired
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Event Event `gorm:"foreignKey:EventID"`
}

// Conversion methods to API DTOs
func (e *Event) ToEventResponse(availableSeats int) *EventResponse {
	return &EventResponse{
		EventID:        e.ID,
		Name:           e.Name,
		Description:    e.Description,
		Venue:          e.Venue,
		City:           e.City,
		Category:       e.Category,
		EventDate:      e.EventDate,
		TotalSeats:     e.TotalSeats,
		AvailableSeats: availableSeats,
		PricePerSeat:   e.PricePerSeat,
		CreatedAt:      e.CreatedAt,
		CreatedBy:      e.CreatedBy,
	}
}

func (h *Hold) ToHoldResponse(totalPrice float64) *HoldResponse {
	return &HoldResponse{
		HoldID:     h.ID,
		EventID:    h.EventID,
		HeldSeats:  h.SeatNumbers,
		ExpiresAt:  h.ExpiresAt,
		TotalPrice: totalPrice,
	}
}

// ===============================
// Repository DTOs (Internal)
// ===============================

// CreateEventRequest represents input for creating an event in repository layer
type CreateEventRequest struct {
	ID           string
	Name         string
	Description  string
	Venue        string
	City         string
	Category     string
	EventDate    time.Time
	TotalSeats   int
	PricePerSeat float64
	CreatedBy    string
}

// UpdateEventRequest represents input for updating an event in repository layer
type UpdateEventRequest struct {
	ID           string
	Name         string
	Description  string
	Venue        string
	City         string
	Category     string
	EventDate    time.Time
	TotalSeats   int
	PricePerSeat float64
}

// EventFilter represents filtering options for repository layer
type EventFilter struct {
	City     string
	DateFrom *time.Time
	DateTo   *time.Time
	Category string
	Name     string
	Limit    int
	Offset   int
}

// CreateHoldRequest represents input for creating a hold in repository layer
type CreateHoldRequest struct {
	ID          string
	UserID      string
	EventID     string
	SeatNumbers []string
	ExpiresAt   time.Time
}

// ===============================
// API DTOs (External)
// ===============================

// CreateEventRequest represents the API request for creating an event
type CreateEventAPIRequest struct {
	Name         string    `json:"name" binding:"required"`
	Description  string    `json:"description"`
	Venue        string    `json:"venue" binding:"required"`
	City         string    `json:"city" binding:"required"`
	Category     string    `json:"category" binding:"required"`
	EventDate    time.Time `json:"event_date" binding:"required"`
	TotalSeats   int       `json:"total_seats" binding:"required,min=1,max=10000"`
	PricePerSeat float64   `json:"price_per_seat" binding:"required,min=0.01"`
}

// ToCreateEventRequest converts API request to repository request
func (r *CreateEventAPIRequest) ToCreateEventRequest(userID string) CreateEventRequest {
	return CreateEventRequest{
		Name:         r.Name,
		Description:  r.Description,
		Venue:        r.Venue,
		City:         r.City,
		Category:     r.Category,
		EventDate:    r.EventDate,
		TotalSeats:   r.TotalSeats,
		PricePerSeat: r.PricePerSeat,
		CreatedBy:    userID,
	}
}

// HoldSeatsRequest represents the API request for holding seats
type HoldSeatsRequest struct {
	SeatNumbers []string `json:"seat_numbers" binding:"required,min=1"`
}

// ToCreateHoldRequest converts API request to repository request
func (r *HoldSeatsRequest) ToCreateHoldRequest(userID, eventID string, expiresAt time.Time) CreateHoldRequest {
	return CreateHoldRequest{
		UserID:      userID,
		EventID:     eventID,
		SeatNumbers: r.SeatNumbers,
		ExpiresAt:   expiresAt,
	}
}

// EventResponse represents event data in API responses
type EventResponse struct {
	EventID              string    `json:"event_id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description,omitempty"`
	Venue                string    `json:"venue"`
	City                 string    `json:"city"`
	Category             string    `json:"category"`
	EventDate            time.Time `json:"event_date"`
	TotalSeats           int       `json:"total_seats"`
	AvailableSeats       int       `json:"available_seats"`
	PricePerSeat         float64   `json:"price_per_seat"`
	AvailableSeatNumbers []string  `json:"available_seat_numbers,omitempty"` // Only in detail view
	CreatedAt            time.Time `json:"created_at"`
	CreatedBy            string    `json:"created_by"`
}

// EventListResponse represents the response for listing events
type EventListResponse struct {
	Events     []EventResponse `json:"events"`
	Pagination Pagination      `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// HoldResponse represents the response for seat hold operations
type HoldResponse struct {
	HoldID     string    `json:"hold_id"`
	EventID    string    `json:"event_id"`
	HeldSeats  []string  `json:"held_seats"`
	ExpiresAt  time.Time `json:"expires_at"`
	TotalPrice float64   `json:"total_price"`
}

// SeatsNotAvailableError represents error when seats are not available
type SeatsNotAvailableError struct {
	UnavailableSeats      []string `json:"unavailable_seats"`
	AvailableAlternatives []string `json:"available_alternatives"`
}

// ErrorResponse represents error responses
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

// HoldDetailsResponse represents hold details for external services
type HoldDetailsResponse struct {
	HoldID     string    `json:"hold_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name,omitempty"` // Optional, requires user service lookup
	EventID    string    `json:"event_id"`
	EventName  string    `json:"event_name"`
	Venue      string    `json:"venue"`
	EventDate  time.Time `json:"event_date"`
	Seats      []string  `json:"seats"`
	TotalPrice float64   `json:"total_price"`
	ExpiresAt  time.Time `json:"expires_at"`
}
