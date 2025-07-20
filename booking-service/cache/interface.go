package cache

import (
	"time"

	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/google/uuid"
)

// CacheRepository defines the interface for booking caching operations
type CacheRepository interface {
	// Booking status caching for SSE
	GetBookingStatus(bookingID uuid.UUID) (*model.BookingStatusUpdate, error)
	SetBookingStatus(bookingID uuid.UUID, status *model.BookingStatusUpdate, ttl time.Duration) error
	InvalidateBookingStatus(bookingID uuid.UUID) error

	// Health check
	Ping() error
}
