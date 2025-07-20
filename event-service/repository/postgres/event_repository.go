package postgres

import (
	"errors"
	"fmt"
	"log"

	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresEventRepository struct {
	db *gorm.DB
}

func NewEventRepository(databaseURL string) (*PostgresEventRepository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(&model.Event{}, &model.Seat{}, &model.Hold{}); err != nil {
		return nil, err
	}

	log.Println("Database connected and Event tables migrated successfully")

	return &PostgresEventRepository{db: db}, nil
}

// Event operations
func (r *PostgresEventRepository) CreateEvent(req model.CreateEventRequest) (*model.Event, error) {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create event
	event := model.Event{
		Name:         req.Name,
		Description:  req.Description,
		Venue:        req.Venue,
		City:         req.City,
		Category:     req.Category,
		EventDate:    req.EventDate,
		TotalSeats:   req.TotalSeats,
		PricePerSeat: req.PricePerSeat,
		CreatedBy:    req.CreatedBy,
	}

	if err := tx.Create(&event).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Generate seats (A1, A2, ... B1, B2, ...)
	seats := r.generateSeats(event.ID, req.TotalSeats)
	if err := tx.CreateInBatches(seats, 100).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &event, nil
}

func (r *PostgresEventRepository) GetEventByID(eventID uuid.UUID) (*model.Event, error) {
	var event model.Event
	if err := r.db.Where("id = ?", eventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}
	return &event, nil
}

func (r *PostgresEventRepository) ListEvents(filter model.EventFilter) ([]model.Event, int, error) {
	var events []model.Event
	var total int64

	query := r.db.Model(&model.Event{})

	// Apply filters
	if filter.City != "" {
		query = query.Where("city ILIKE ?", "%"+filter.City+"%")
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}
	if filter.DateFrom != nil {
		query = query.Where("event_date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("event_date <= ?", *filter.DateTo)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and get results
	if err := query.Offset(filter.Offset).Limit(filter.Limit).Order("event_date ASC").Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, int(total), nil
}

func (r *PostgresEventRepository) UpdateEvent(req model.UpdateEventRequest) (*model.Event, error) {
	var event model.Event
	if err := r.db.Where("id = ?", req.ID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	// Update fields
	event.Name = req.Name
	event.Description = req.Description
	event.Venue = req.Venue
	event.City = req.City
	event.Category = req.Category
	event.EventDate = req.EventDate
	event.TotalSeats = req.TotalSeats
	event.PricePerSeat = req.PricePerSeat

	if err := r.db.Save(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

func (r *PostgresEventRepository) DeleteEvent(eventID uuid.UUID) error {
	result := r.db.Delete(&model.Event{}, eventID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

// Seat operations
func (r *PostgresEventRepository) GetAvailableSeats(eventID uuid.UUID) ([]string, error) {
	var seats []string
	query := `
		SELECT seat_number FROM seats s
		LEFT JOIN holds h ON s.hold_id = h.id
		WHERE s.event_id = ? 
		AND (s.status = 'available' 
			 OR (s.status = 'held' AND h.expires_at < NOW()))
		ORDER BY seat_number
	`
	if err := r.db.Raw(query, eventID).Scan(&seats).Error; err != nil {
		return nil, err
	}
	return seats, nil
}

func (r *PostgresEventRepository) GetAvailableSeatCount(eventID uuid.UUID) (int, error) {
	var count int64
	query := `
		SELECT COUNT(*) FROM seats s
		LEFT JOIN holds h ON s.hold_id = h.id
		WHERE s.event_id = ? 
		AND (s.status = 'available' 
			 OR (s.status = 'held' AND h.expires_at < NOW()))
	`
	if err := r.db.Raw(query, eventID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *PostgresEventRepository) CheckSeatsAvailability(eventID uuid.UUID, seatNumbers []string) ([]string, []string, error) {
	var unavailableSeats []string
	query := `
		SELECT seat_number FROM seats s
		LEFT JOIN holds h ON s.hold_id = h.id
		WHERE s.event_id = ? AND s.seat_number = ANY(?)
		AND s.status != 'available' 
		AND NOT (s.status = 'held' AND h.expires_at < NOW())
	`
	if err := r.db.Raw(query, eventID, seatNumbers).Scan(&unavailableSeats).Error; err != nil {
		return nil, nil, err
	}

	// Calculate available seats
	unavailableMap := make(map[string]bool)
	for _, seat := range unavailableSeats {
		unavailableMap[seat] = true
	}

	var availableSeats []string
	for _, seat := range seatNumbers {
		if !unavailableMap[seat] {
			availableSeats = append(availableSeats, seat)
		}
	}

	return availableSeats, unavailableSeats, nil
}

// Hold operations
func (r *PostgresEventRepository) CreateHold(req model.CreateHoldRequest) (*model.Hold, error) {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check seat availability
	_, unavailable, err := r.CheckSeatsAvailability(req.EventID, req.SeatNumbers)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(unavailable) > 0 {
		tx.Rollback()
		return nil, fmt.Errorf("seats not available: %v", unavailable)
	}

	// Create hold
	hold := model.Hold{
		UserID:      req.UserID,
		EventID:     req.EventID,
		SeatNumbers: req.SeatNumbers,
		ExpiresAt:   req.ExpiresAt,
		Status:      "active",
	}

	if err := tx.Create(&hold).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Update seat status
	if err := tx.Model(&model.Seat{}).
		Where("event_id = ? AND seat_number IN ?", req.EventID, req.SeatNumbers).
		Updates(map[string]interface{}{
			"status":  "held",
			"hold_id": hold.ID,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &hold, nil
}

func (r *PostgresEventRepository) GetHoldByID(holdID uuid.UUID) (*model.Hold, error) {
	var hold model.Hold
	if err := r.db.Where("id = ?", holdID).First(&hold).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hold not found")
		}
		return nil, err
	}
	return &hold, nil
}

func (r *PostgresEventRepository) ReleaseHold(holdID uuid.UUID) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get hold
	var hold model.Hold
	if err := tx.Where("id = ?", holdID).First(&hold).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("hold not found")
		}
		return err
	}

	// Update seat status back to available
	if err := tx.Model(&model.Seat{}).
		Where("event_id = ? AND seat_number IN ?", hold.EventID, hold.SeatNumbers).
		Updates(map[string]interface{}{
			"status":  "available",
			"hold_id": nil,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update hold status
	if err := tx.Model(&hold).Update("status", "expired").Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (r *PostgresEventRepository) ConfirmHold(holdID uuid.UUID) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get hold
	var hold model.Hold
	if err := tx.Where("id = ?", holdID).First(&hold).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("hold not found")
		}
		return err
	}

	// Update seat status to booked
	if err := tx.Model(&model.Seat{}).
		Where("event_id = ? AND seat_number IN ?", hold.EventID, hold.SeatNumbers).
		Update("status", "booked").Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update hold status
	if err := tx.Model(&hold).Update("status", "confirmed").Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (r *PostgresEventRepository) CleanupExpiredHolds() error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get expired holds
	var expiredHolds []model.Hold
	if err := tx.Where("expires_at < NOW() AND status = 'active'").Find(&expiredHolds).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, hold := range expiredHolds {
		// Release seats
		if err := tx.Model(&model.Seat{}).
			Where("event_id = ? AND seat_number IN ?", hold.EventID, hold.SeatNumbers).
			Updates(map[string]interface{}{
				"status":  "available",
				"hold_id": nil,
			}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Update hold status
		if err := tx.Model(&hold).Update("status", "expired").Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func (r *PostgresEventRepository) GetDB() *gorm.DB {
	return r.db
}

// Helper function to generate seats
func (r *PostgresEventRepository) generateSeats(eventID uuid.UUID, totalSeats int) []model.Seat {
	var seats []model.Seat
	seatCount := 0
	row := 'A'

	for seatCount < totalSeats {
		seatNum := 1
		for seatNum <= 50 && seatCount < totalSeats { // Max 50 seats per row
			seatNumber := fmt.Sprintf("%c%d", row, seatNum)
			seats = append(seats, model.Seat{
				EventID:    eventID,
				SeatNumber: seatNumber,
				Status:     "available",
			})
			seatNum++
			seatCount++
		}
		row++
	}

	return seats
}
