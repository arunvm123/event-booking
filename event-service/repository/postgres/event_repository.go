package postgres

import (
	"errors"
	"fmt"
	"log"

	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/google/uuid"
	"github.com/lib/pq"
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

	// Create performance indexes for high-load scenarios
	if err := createPerformanceIndexes(db); err != nil {
		log.Printf("Warning: Failed to create some performance indexes: %v", err)
		// Don't fail startup, just warn - indexes can be created manually
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
		ID:           req.ID,
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

func (r *PostgresEventRepository) GetEventByID(eventID string) (*model.Event, error) {
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

func (r *PostgresEventRepository) DeleteEvent(eventID string) error {
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
func (r *PostgresEventRepository) GetAvailableSeats(eventID string) ([]string, error) {
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

func (r *PostgresEventRepository) GetAvailableSeatCount(eventID string) (int, error) {
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

func (r *PostgresEventRepository) GetAvailableSeatNumbers(eventID string) ([]string, error) {
	return r.GetAvailableSeats(eventID)
}

func (r *PostgresEventRepository) CheckSeatsAvailability(eventID string, seatNumbers []string) error {
	var unavailableSeats []string
	query := `
		SELECT seat_number FROM seats s
		LEFT JOIN holds h ON s.hold_id = h.id
		WHERE s.event_id = ? AND s.seat_number = ANY(?)
		AND s.status != 'available' 
		AND NOT (s.status = 'held' AND h.expires_at < NOW())
	`
	if err := r.db.Raw(query, eventID, pq.Array(seatNumbers)).Scan(&unavailableSeats).Error; err != nil {
		return err
	}

	if len(unavailableSeats) > 0 {
		return fmt.Errorf("seats not available")
	}

	return nil
}

// CheckSeatsExist validates that all requested seat numbers exist for the given event
func (r *PostgresEventRepository) CheckSeatsExist(eventID string, seatNumbers []string) error {
	var existingSeats []string
	query := `
		SELECT seat_number FROM seats 
		WHERE event_id = ? AND seat_number = ANY(?)
	`
	if err := r.db.Raw(query, eventID, pq.Array(seatNumbers)).Scan(&existingSeats).Error; err != nil {
		return fmt.Errorf("failed to check seat existence: %w", err)
	}

	// Check if all requested seats exist
	if len(existingSeats) != len(seatNumbers) {
		// Find which seats don't exist
		existingSet := make(map[string]bool)
		for _, seat := range existingSeats {
			existingSet[seat] = true
		}

		var nonExistentSeats []string
		for _, seat := range seatNumbers {
			if !existingSet[seat] {
				nonExistentSeats = append(nonExistentSeats, seat)
			}
		}

		return fmt.Errorf("seat numbers do not exist: %v", nonExistentSeats)
	}

	return nil
}

// Hold operations
func (r *PostgresEventRepository) CreateHold(req model.CreateHoldRequest) (*model.Hold, error) {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// First check if seats exist
	err := r.CheckSeatsExist(req.EventID, req.SeatNumbers)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Then check seat availability
	err = r.CheckSeatsAvailability(req.EventID, req.SeatNumbers)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create hold
	hold := model.Hold{
		ID:          req.ID,
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
		Where("event_id = ? AND seat_number IN (?)", req.EventID, req.SeatNumbers).
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

func (r *PostgresEventRepository) GetHoldByID(holdID string) (*model.Hold, error) {
	var hold model.Hold
	if err := r.db.Where("id = ?", holdID).First(&hold).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hold not found")
		}
		return nil, err
	}
	return &hold, nil
}

func (r *PostgresEventRepository) ReleaseHold(holdID string) error {
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
		Where("event_id = ? AND seat_number IN (?)", hold.EventID, hold.SeatNumbers).
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

func (r *PostgresEventRepository) ConfirmHold(holdID string) error {
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
		Where("event_id = ? AND seat_number IN (?)", hold.EventID, hold.SeatNumbers).
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
			Where("event_id = ? AND seat_number IN (?)", hold.EventID, hold.SeatNumbers).
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
// Pattern: A1-A500, B1-B500, ..., Z1-Z500, AA1-AA500, AB1-AB500, etc.
func (r *PostgresEventRepository) generateSeats(eventID string, totalSeats int) []model.Seat {
	var seats []model.Seat
	seatCount := 0
	rowIndex := 0

	for seatCount < totalSeats {
		seatNum := 1
		rowName := generateRowName(rowIndex)

		// Generate up to 500 seats per row (configurable)
		for seatNum <= 500 && seatCount < totalSeats {
			seatNumber := fmt.Sprintf("%s%d", rowName, seatNum)
			seats = append(seats, model.Seat{
				ID:         uuid.New().String(),
				EventID:    eventID,
				SeatNumber: seatNumber,
				Status:     "available",
			})
			seatNum++
			seatCount++
		}
		rowIndex++
	}

	return seats
}

// generateRowName converts row index to Excel-style column names
// 0 -> A, 1 -> B, ..., 25 -> Z, 26 -> AA, 27 -> AB, etc.
func generateRowName(index int) string {
	result := ""

	for {
		result = string(rune('A'+(index%26))) + result
		index = index / 26
		if index == 0 {
			break
		}
		index-- // Adjust for the fact that there's no "zero" letter
	}

	return result
}

// createPerformanceIndexes creates critical indexes for high-performance operations
func createPerformanceIndexes(db *gorm.DB) error {
	// Critical indexes for CheckSeatsAvailability performance
	indexes := []string{
		// 1. MOST CRITICAL: Event + Seat lookup (fixes 80% of performance issues)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_seats_event_seat_status 
		 ON seats (event_id, seat_number, status)`,

		// 2. HOLDS JOIN: Speeds up the LEFT JOIN operation
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holds_id_expires 
		 ON holds (id, expires_at)`,

		// 3. COMPLETE OPTIMIZATION: Covers the full CheckSeatsAvailability query
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_seats_availability_check 
		 ON seats (event_id, seat_number, status, hold_id)`,

		// 4. Event listing performance - Date filtering
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_event_date 
		 ON events (event_date)`,

		// 5. Event listing performance - City filtering
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_city 
		 ON events (city)`,

		// 6. Event listing performance - Category filtering
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_category 
		 ON events (category)`,

		// 7. Composite index for common filter combinations
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_date_city 
		 ON events (event_date, city)`,

		// 8. Hold cleanup performance - Expired holds
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holds_expires_at 
		 ON holds (expires_at) WHERE expires_at IS NOT NULL`,

		// 9. Seat availability count optimization
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_seats_event_status 
		 ON seats (event_id, status)`,

		// 10. User's holds lookup
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_holds_user_id 
		 ON holds (user_id)`,
	}

	log.Println("Creating performance indexes for high-load scenarios...")

	for i, indexSQL := range indexes {
		log.Printf("Creating index %d/%d...", i+1, len(indexes))

		// Execute the index creation
		if err := db.Exec(indexSQL).Error; err != nil {
			// Log warning but continue with other indexes
			log.Printf("Warning: Failed to create index %d: %v", i+1, err)
			log.Printf("Index SQL: %s", indexSQL)
		}
	}

	log.Println("Performance indexes creation completed")
	return nil
}

// CreatePerformanceIndexes manually creates performance indexes
// Call this if you want to create indexes after startup
func (r *PostgresEventRepository) CreatePerformanceIndexes() error {
	return createPerformanceIndexes(r.db)
}

// VerifyIndexes checks if critical performance indexes exist
func (r *PostgresEventRepository) VerifyIndexes() error {
	criticalIndexes := []string{
		"idx_seats_event_seat_status",
		"idx_holds_id_expires",
		"idx_seats_availability_check",
		"idx_events_event_date",
		"idx_events_city",
	}

	for _, indexName := range criticalIndexes {
		var exists bool
		query := `SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = ? AND tablename IN ('seats', 'holds', 'events')
		)`

		if err := r.db.Raw(query, indexName).Scan(&exists).Error; err != nil {
			return fmt.Errorf("failed to check index %s: %w", indexName, err)
		}

		if !exists {
			log.Printf("Warning: Critical index %s does not exist", indexName)
		} else {
			log.Printf("✓ Index %s exists", indexName)
		}
	}

	return nil
}
