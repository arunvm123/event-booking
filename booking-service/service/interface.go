package service

import (
	"github.com/google/uuid"
)

// EventService defines the interface for communicating with the Event Service
type EventService interface {
	// GetHoldDetails retrieves hold information from the event service
	GetHoldDetails(holdID uuid.UUID) (*HoldDetails, error)

	// ConfirmHold confirms a hold (converts it to booking) in the event service
	ConfirmHold(holdID uuid.UUID) error

	// ReleaseHold releases a hold in the event service
	ReleaseHold(holdID uuid.UUID) error
}

// HoldDetails represents hold information from the event service
type HoldDetails struct {
	HoldID     uuid.UUID `json:"hold_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	EventID    uuid.UUID `json:"event_id"`
	EventName  string    `json:"event_name"`
	Venue      string    `json:"venue"`
	EventDate  string    `json:"event_date"`
	Seats      []string  `json:"seats"`
	TotalPrice float64   `json:"total_price"`
	ExpiresAt  string    `json:"expires_at"`
}
