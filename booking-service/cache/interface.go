package cache

import (
	"time"

	"github.com/arunvm123/eventbooking/booking-service/model"
)

// CacheRepository defines the interface for booking caching operations
type CacheRepository interface {
	// Booking status caching for SSE
	GetBookingStatus(bookingID string) (*model.BookingStatusUpdate, error)
	SetBookingStatus(bookingID string, status *model.BookingStatusUpdate, ttl time.Duration) error
	InvalidateBookingStatus(bookingID string) error

	// Health check
	Ping() error
}
