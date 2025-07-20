package repository

import (
	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BookingRepository defines the interface for booking data operations
type BookingRepository interface {
	// Booking operations
	CreateBooking(req model.CreateBookingRequest) (*model.Booking, error)
	GetBookingByID(bookingID uuid.UUID) (*model.Booking, error)
	GetBookingByHoldID(holdID uuid.UUID) (*model.Booking, error)
	UpdateBookingStatus(req model.UpdateBookingStatusRequest) error
	ListUserBookings(filter model.BookingFilter) ([]model.Booking, int, error)

	// Health check
	GetDB() *gorm.DB
}
