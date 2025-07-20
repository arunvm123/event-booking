package cache

import (
	"time"

	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/google/uuid"
)

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Seat availability caching
	GetAvailableSeats(eventID uuid.UUID) ([]string, error)
	SetAvailableSeats(eventID uuid.UUID, seats []string, ttl time.Duration) error
	InvalidateAvailableSeats(eventID uuid.UUID) error

	// Seat count caching
	GetAvailableSeatCount(eventID uuid.UUID) (int, error)
	SetAvailableSeatCount(eventID uuid.UUID, count int, ttl time.Duration) error
	InvalidateAvailableSeatCount(eventID uuid.UUID) error

	// Event details caching
	GetEvent(eventID uuid.UUID) (*model.Event, error)
	SetEvent(eventID uuid.UUID, event *model.Event, ttl time.Duration) error
	InvalidateEvent(eventID uuid.UUID) error

	// Event list caching with filters
	GetEventList(filterKey string) (*model.EventListResponse, error)
	SetEventList(filterKey string, response *model.EventListResponse, ttl time.Duration) error
	InvalidateEventList(pattern string) error

	// Health check
	Ping() error

	// Cache invalidation patterns
	InvalidateEventRelatedCache(eventID uuid.UUID) error
}
