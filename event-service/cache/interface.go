package cache

import (
	"time"

	"github.com/arunvm123/eventbooking/event-service/model"
)

type CacheRepository interface {
	// Available seat operations
	GetAvailableSeats(eventID string) ([]string, error)
	SetAvailableSeats(eventID string, seats []string, ttl time.Duration) error
	InvalidateAvailableSeats(eventID string) error

	// Available seat count operations
	GetAvailableSeatCount(eventID string) (int, error)
	SetAvailableSeatCount(eventID string, count int, ttl time.Duration) error
	InvalidateAvailableSeatCount(eventID string) error

	// Event operations
	GetEvent(eventID string) (*model.Event, error)
	SetEvent(eventID string, event *model.Event, ttl time.Duration) error
	InvalidateEvent(eventID string) error

	// Event list operations
	GetEventList(filterKey string) (*model.EventListResponse, error)
	SetEventList(filterKey string, response *model.EventListResponse, ttl time.Duration) error
	InvalidateEventList(pattern string) error

	// Health check
	Ping() error

	// Cache invalidation patterns
	InvalidateEventRelatedCache(eventID string) error
}
