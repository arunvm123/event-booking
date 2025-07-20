package repository

import (
	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventRepository defines the interface for event data operations
type EventRepository interface {
	// Event operations
	CreateEvent(req model.CreateEventRequest) (*model.Event, error)
	GetEventByID(eventID uuid.UUID) (*model.Event, error)
	ListEvents(filter model.EventFilter) ([]model.Event, int, error)
	UpdateEvent(req model.UpdateEventRequest) (*model.Event, error)
	DeleteEvent(eventID uuid.UUID) error

	// Seat operations
	GetAvailableSeats(eventID uuid.UUID) ([]string, error)
	GetAvailableSeatCount(eventID uuid.UUID) (int, error)
	CheckSeatsAvailability(eventID uuid.UUID, seatNumbers []string) ([]string, []string, error) // available, unavailable

	// Hold operations
	CreateHold(req model.CreateHoldRequest) (*model.Hold, error)
	GetHoldByID(holdID uuid.UUID) (*model.Hold, error)
	ReleaseHold(holdID uuid.UUID) error
	ConfirmHold(holdID uuid.UUID) error
	CleanupExpiredHolds() error

	// Health check
	GetDB() *gorm.DB
}
