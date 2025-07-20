package postgres

import (
	"fmt"

	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresBookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(databaseURL string) (*PostgresBookingRepository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the booking table
	if err := db.AutoMigrate(&model.Booking{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &PostgresBookingRepository{db: db}, nil
}

// CreateBooking creates a new booking record
func (r *PostgresBookingRepository) CreateBooking(req model.CreateBookingRequest) (*model.Booking, error) {
	booking := &model.Booking{
		UserID:        req.UserID,
		UserEmail:     req.UserEmail,
		UserName:      req.UserName,
		EventID:       req.EventID,
		EventName:     req.EventName,
		Venue:         req.Venue,
		EventDate:     req.EventDate,
		Seats:         req.Seats,
		TotalAmount:   req.TotalAmount,
		Status:        "processing",
		PaymentStatus: "pending",
		HoldID:        req.HoldID,
	}

	if err := r.db.Create(booking).Error; err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	return booking, nil
}

// GetBookingByID retrieves a booking by its ID
func (r *PostgresBookingRepository) GetBookingByID(bookingID uuid.UUID) (*model.Booking, error) {
	var booking model.Booking
	err := r.db.Where("id = ?", bookingID).First(&booking).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("booking not found")
		}
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	return &booking, nil
}

// GetBookingByHoldID retrieves a booking by hold ID
func (r *PostgresBookingRepository) GetBookingByHoldID(holdID uuid.UUID) (*model.Booking, error) {
	var booking model.Booking
	err := r.db.Where("hold_id = ?", holdID).First(&booking).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("booking not found")
		}
		return nil, fmt.Errorf("failed to get booking by hold ID: %w", err)
	}

	return &booking, nil
}

// UpdateBookingStatus updates the status of a booking
func (r *PostgresBookingRepository) UpdateBookingStatus(req model.UpdateBookingStatusRequest) error {
	updates := map[string]interface{}{
		"status":         req.Status,
		"payment_status": req.PaymentStatus,
	}

	if req.ErrorMessage != nil {
		updates["error_message"] = *req.ErrorMessage
	}

	if req.ConfirmedAt != nil {
		updates["confirmed_at"] = *req.ConfirmedAt
	}

	if req.FailedAt != nil {
		updates["failed_at"] = *req.FailedAt
	}

	err := r.db.Model(&model.Booking{}).Where("id = ?", req.BookingID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update booking status: %w", err)
	}

	return nil
}

// ListUserBookings retrieves bookings for a specific user with filtering
func (r *PostgresBookingRepository) ListUserBookings(filter model.BookingFilter) ([]model.Booking, int, error) {
	var bookings []model.Booking
	var total int64

	query := r.db.Model(&model.Booking{}).Where("user_id = ?", filter.UserID)

	// Apply status filter if specified
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count bookings: %w", err)
	}

	// Apply pagination and ordering
	err := query.Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&bookings).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list bookings: %w", err)
	}

	return bookings, int(total), nil
}

// GetDB returns the database instance for health checks
func (r *PostgresBookingRepository) GetDB() *gorm.DB {
	return r.db
}
