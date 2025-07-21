package repository

import (
	"github.com/arunvm123/eventbooking/event-service/model"
	"gorm.io/gorm"
)

type EventRepository interface {
	// Event operations
	CreateEvent(req model.CreateEventRequest) (*model.Event, error)
	GetEventByID(id string) (*model.Event, error)
	UpdateEvent(req model.UpdateEventRequest) (*model.Event, error)
	DeleteEvent(id string) error
	ListEvents(filter model.EventFilter) ([]model.Event, int, error)

	// Seat operations
	GetAvailableSeatCount(eventID string) (int, error)
	GetAvailableSeatNumbers(eventID string) ([]string, error)
	GetAvailableSeats(eventID string) ([]string, error)
	CheckSeatsAvailability(eventID string, seatNumbers []string) error

	// Hold operations
	CreateHold(req model.CreateHoldRequest) (*model.Hold, error)
	GetHoldByID(id string) (*model.Hold, error)
	ReleaseHold(id string) error
	ConfirmHold(id string) error
	CleanupExpiredHolds() error

	// Database access for health checks
	GetDB() *gorm.DB
}
