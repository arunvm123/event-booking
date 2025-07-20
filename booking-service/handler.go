package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/arunvm123/eventbooking/booking-service/cache"
	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/arunvm123/eventbooking/booking-service/repository"
	"github.com/arunvm123/eventbooking/booking-service/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type BookingHandler struct {
	repo         repository.BookingRepository
	cache        cache.CacheRepository
	kafkaWriter  *kafka.Writer
	eventService service.EventService
}

func NewBookingHandler(repo repository.BookingRepository, cache cache.CacheRepository, kafkaWriter *kafka.Writer, eventService service.EventService) *BookingHandler {
	return &BookingHandler{
		repo:         repo,
		cache:        cache,
		kafkaWriter:  kafkaWriter,
		eventService: eventService,
	}
}

// SubmitBooking handles booking submission and queues for async processing
func (h *BookingHandler) SubmitBooking(c *gin.Context) {
	var req model.SubmitBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
		return
	}

	// Get user info from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in token",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
		})
		return
	}

	userEmail, _ := c.Get("user_email")
	userEmailStr, _ := userEmail.(string)

	// Check if booking already exists for this hold
	existingBooking, err := h.repo.GetBookingByHoldID(req.HoldID)
	if err == nil && existingBooking != nil {
		// Return existing booking
		response := model.BookingResponse{
			BookingID:     existingBooking.ID,
			Status:        existingBooking.Status,
			Message:       "Booking already exists for this hold",
			EstimatedTime: "Already processed",
			StatusURL:     fmt.Sprintf("/api/booking/%s/status", existingBooking.ID.String()),
			StreamURL:     fmt.Sprintf("/api/booking/%s/stream", existingBooking.ID.String()),
		}
		c.JSON(http.StatusAccepted, response)
		return
	}

	// Get hold details from event service
	holdDetails, err := h.eventService.GetHoldDetails(req.HoldID)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_hold",
			Message: "Failed to validate hold: " + err.Error(),
		})
		return
	}

	// Parse event date
	eventDate, err := time.Parse(time.RFC3339, holdDetails.EventDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid event date format",
		})
		return
	}

	// Create booking record with processing status
	createReq := model.CreateBookingRequest{
		UserID:        userUUID,
		UserEmail:     userEmailStr,
		UserName:      holdDetails.UserName,
		EventID:       holdDetails.EventID,
		EventName:     holdDetails.EventName,
		Venue:         holdDetails.Venue,
		EventDate:     eventDate,
		Seats:         holdDetails.Seats,
		TotalAmount:   req.PaymentInfo.Amount,
		HoldID:        req.HoldID,
		PaymentMethod: req.PaymentInfo.PaymentMethod,
	}

	booking, err := h.repo.CreateBooking(createReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create booking",
		})
		return
	}

	// Send to Kafka for async processing
	kafkaMsg := model.BookingRequest{
		BookingID:   booking.ID,
		UserID:      userUUID,
		UserEmail:   userEmailStr,
		UserName:    holdDetails.UserName,
		HoldID:      req.HoldID,
		EventID:     holdDetails.EventID,
		EventName:   holdDetails.EventName,
		Venue:       holdDetails.Venue,
		EventDate:   eventDate,
		Seats:       holdDetails.Seats,
		PaymentInfo: req.PaymentInfo,
		Timestamp:   time.Now(),
	}

	msgBytes, _ := json.Marshal(kafkaMsg)
	h.kafkaWriter.WriteMessages(c.Request.Context(),
		kafka.Message{
			Key:   []byte(booking.ID.String()),
			Value: msgBytes,
		})

	// Cache initial status
	statusUpdate := &model.BookingStatusUpdate{
		BookingID: booking.ID,
		Status:    "PROCESSING",
		Message:   "Booking submitted for processing",
		UpdatedAt: time.Now(),
	}
	h.cache.SetBookingStatus(booking.ID, statusUpdate, 24*time.Hour)

	// Return immediate response
	response := model.BookingResponse{
		BookingID:     booking.ID,
		Status:        "PROCESSING",
		Message:       "Booking is being processed",
		EstimatedTime: "2-3 minutes",
		StatusURL:     fmt.Sprintf("/api/booking/%s/status", booking.ID.String()),
		StreamURL:     fmt.Sprintf("/api/booking/%s/stream", booking.ID.String()),
	}

	c.JSON(http.StatusAccepted, response)
}

// GetBookingStatus returns the current status of a booking
func (h *BookingHandler) GetBookingStatus(c *gin.Context) {
	bookingIDStr := c.Param("bookingId")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid booking ID format",
		})
		return
	}

	booking, err := h.repo.GetBookingByID(bookingID)
	if err != nil {
		if err.Error() == "booking not found" {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Error:   "not_found",
				Message: "Booking not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve booking",
		})
		return
	}

	response := booking.ToBookingStatusResponse()
	c.JSON(http.StatusOK, response)
}

// StreamBookingStatus provides Server-Sent Events for real-time booking updates
func (h *BookingHandler) StreamBookingStatus(c *gin.Context) {
	bookingIDStr := c.Param("bookingId")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid booking ID format",
		})
		return
	}

	// Verify booking exists
	booking, err := h.repo.GetBookingByID(bookingID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{
			Error:   "not_found",
			Message: "Booking not found",
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Send initial status
	statusUpdate := &model.BookingStatusUpdate{
		BookingID: booking.ID,
		Status:    booking.Status,
		Message:   fmt.Sprintf("Current status: %s", booking.Status),
		UpdatedAt: time.Now(),
	}

	eventData, _ := json.Marshal(statusUpdate)
	c.SSEvent("status", string(eventData))
	c.Writer.Flush()

	// If booking is final, close stream
	if booking.Status == "confirmed" || booking.Status == "failed" {
		finalData, _ := json.Marshal(map[string]interface{}{
			"booking_id":   booking.ID,
			"final_status": booking.Status,
		})
		c.SSEvent("complete", string(finalData))
		return
	}

	// Keep connection alive and poll for updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for status updates
			updated, err := h.repo.GetBookingByID(bookingID)
			if err != nil {
				continue
			}

			if updated.Status != booking.Status {
				booking = updated
				statusUpdate := &model.BookingStatusUpdate{
					BookingID: booking.ID,
					Status:    booking.Status,
					Message:   fmt.Sprintf("Status updated to: %s", booking.Status),
					UpdatedAt: time.Now(),
				}

				eventData, _ := json.Marshal(statusUpdate)
				c.SSEvent("status", string(eventData))
				c.Writer.Flush()

				// Close stream if final status
				if booking.Status == "confirmed" || booking.Status == "failed" {
					finalData, _ := json.Marshal(map[string]interface{}{
						"booking_id":   booking.ID,
						"final_status": booking.Status,
					})
					c.SSEvent("complete", string(finalData))
					return
				}
			}

		case <-c.Request.Context().Done():
			return
		}
	}
}

// ListUserBookings returns all bookings for the authenticated user
func (h *BookingHandler) ListUserBookings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{
			Error:   "unauthorized",
			Message: "User ID not found in token",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Invalid user ID format",
		})
		return
	}

	filter := model.BookingFilter{
		UserID: userUUID,
		Limit:  50,
		Offset: 0,
	}

	bookings, total, err := h.repo.ListUserBookings(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve bookings",
		})
		return
	}

	var bookingSummaries []model.UserBookingSummary
	for _, booking := range bookings {
		bookingSummaries = append(bookingSummaries, booking.ToUserBookingSummary())
	}

	response := model.UserBookingsResponse{
		Bookings: bookingSummaries,
		Total:    total,
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck handles health check endpoint
func (h *BookingHandler) HealthCheck(c *gin.Context) {
	// Check database connection
	sqlDB, err := h.repo.GetDB().DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database connection failed",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, model.ErrorResponse{
			Error:   "service_unavailable",
			Message: "Database ping failed",
		})
		return
	}

	response := model.HealthResponse{
		Status:    "healthy",
		Service:   "booking-service",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to get hold details from event service
func (h *BookingHandler) getHoldDetails(holdID uuid.UUID) (*HoldData, error) {
	// Mock implementation - in real scenario, call event service API
	// For now, return mock data
	return &HoldData{
		UserName:  "John Doe",
		EventID:   uuid.New(),
		EventName: "Rock Concert 2025",
		Venue:     "Madison Square Garden",
		EventDate: time.Now().Add(30 * 24 * time.Hour),
		Seats:     []string{"A1", "A2", "A3"},
	}, nil
}

type HoldData struct {
	UserName  string
	EventID   uuid.UUID
	EventName string
	Venue     string
	EventDate time.Time
	Seats     []string
}
