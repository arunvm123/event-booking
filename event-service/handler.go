package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/arunvm123/eventbooking/event-service/cache"
	"github.com/arunvm123/eventbooking/event-service/cache/redis"
	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/arunvm123/eventbooking/event-service/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventHandler struct {
	repo  repository.EventRepository
	cache cache.CacheRepository
}

func NewEventHandler(repo repository.EventRepository, cache cache.CacheRepository) *EventHandler {
	return &EventHandler{
		repo:  repo,
		cache: cache,
	}
}

// CreateEvent handles event creation
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req model.CreateEventAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
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

	// Create event
	event, err := h.repo.CreateEvent(req.ToCreateEventRequest(userUUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create event",
		})
		return
	}

	// Invalidate event list caches since new event was created
	h.cache.InvalidateEventList("*")

	// Get available seat count for response
	availableSeats, err := h.repo.GetAvailableSeatCount(event.ID)
	if err != nil {
		availableSeats = event.TotalSeats // fallback
	}

	response := event.ToEventResponse(availableSeats)
	c.JSON(http.StatusCreated, response)
}

// GetEvent handles retrieving a single event by ID
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid event ID format",
		})
		return
	}

	// Try to get event from cache first
	event, err := h.cache.GetEvent(eventID)
	if err != nil || event == nil {
		// Cache miss, get from database
		event, err = h.repo.GetEventByID(eventID)
		if err != nil {
			if err.Error() == "event not found" {
				c.JSON(http.StatusNotFound, model.ErrorResponse{
					Error:   "not_found",
					Message: "Event not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, model.ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to retrieve event",
			})
			return
		}

		// Cache the event for 5 minutes
		h.cache.SetEvent(eventID, event, 5*time.Minute)
	}

	// Try to get available seat count from cache first
	availableSeats, err := h.cache.GetAvailableSeatCount(eventID)
	if err != nil || availableSeats == -1 {
		// Cache miss, get from database
		availableSeats, err = h.repo.GetAvailableSeatCount(eventID)
		if err != nil {
			availableSeats = 0
		} else {
			// Cache the seat count for 30 seconds (more frequent updates)
			h.cache.SetAvailableSeatCount(eventID, availableSeats, 30*time.Second)
		}
	}

	response := event.ToEventResponse(availableSeats)

	// Try to get available seat numbers from cache first
	seatNumbers, err := h.cache.GetAvailableSeats(eventID)
	if err != nil || seatNumbers == nil {
		// Cache miss, get from database
		seatNumbers, err = h.repo.GetAvailableSeats(eventID)
		if err == nil && seatNumbers != nil {
			// Cache seat numbers for 30 seconds
			h.cache.SetAvailableSeats(eventID, seatNumbers, 30*time.Second)
			response.AvailableSeatNumbers = seatNumbers
		}
	} else {
		response.AvailableSeatNumbers = seatNumbers
	}

	c.JSON(http.StatusOK, response)
}

// ListEvents handles event listing with filtering and pagination
func (h *EventHandler) ListEvents(c *gin.Context) {
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	filter := model.EventFilter{
		City:     c.Query("city"),
		Category: c.Query("category"),
		Name:     c.Query("name"),
		Limit:    limit,
		Offset:   offset,
	}

	// Parse date filters
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.DateFrom = &dateFrom
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			filter.DateTo = &dateTo
		}
	}

	// Try to get cached event list first
	filterKey := redis.GenerateFilterKey(filter)
	cachedResponse, err := h.cache.GetEventList(filterKey)
	if err == nil && cachedResponse != nil {
		// Cache hit
		c.JSON(http.StatusOK, cachedResponse)
		return
	}

	// Cache miss, get from database
	events, total, err := h.repo.ListEvents(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve events",
		})
		return
	}

	// Convert to response format
	var eventResponses []model.EventResponse
	for _, event := range events {
		// Try to get available seat count from cache first
		availableSeats, err := h.cache.GetAvailableSeatCount(event.ID)
		if err != nil || availableSeats == -1 {
			// Cache miss, get from database
			availableSeats, err = h.repo.GetAvailableSeatCount(event.ID)
			if err != nil {
				availableSeats = 0
			} else {
				// Cache the seat count for 30 seconds
				h.cache.SetAvailableSeatCount(event.ID, availableSeats, 30*time.Second)
			}
		}
		eventResponses = append(eventResponses, *event.ToEventResponse(availableSeats))
	}

	response := model.EventListResponse{
		Events: eventResponses,
		Pagination: model.Pagination{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasMore: offset+limit < total,
		},
	}

	// Cache the response for 2 minutes
	h.cache.SetEventList(filterKey, &response, 2*time.Minute)

	c.JSON(http.StatusOK, response)
}

// HoldSeats handles seat holding requests
func (h *EventHandler) HoldSeats(c *gin.Context) {
	eventIDStr := c.Param("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid event ID format",
		})
		return
	}

	var req model.HoldSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context
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

	// Hold expires in 15 minutes
	expiresAt := time.Now().Add(15 * time.Minute)

	// Create hold
	hold, err := h.repo.CreateHold(req.ToCreateHoldRequest(userUUID, eventID, expiresAt))
	if err != nil {
		if err.Error() == "seats not available" {
			c.JSON(http.StatusConflict, model.ErrorResponse{
				Error:   "seats_unavailable",
				Message: "Some requested seats are not available",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to hold seats",
		})
		return
	}

	// Invalidate seat-related caches since seats were held
	h.cache.InvalidateAvailableSeats(eventID)
	h.cache.InvalidateAvailableSeatCount(eventID)

	// Get event to calculate total price
	event, err := h.repo.GetEventByID(eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve event details",
		})
		return
	}

	totalPrice := event.PricePerSeat * float64(len(req.SeatNumbers))
	response := hold.ToHoldResponse(totalPrice)

	c.JSON(http.StatusCreated, response)
}

// ReleaseHold handles releasing a seat hold
func (h *EventHandler) ReleaseHold(c *gin.Context) {
	holdIDStr := c.Param("holdId")
	holdID, err := uuid.Parse(holdIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid hold ID format",
		})
		return
	}

	// Get hold first to know which event to invalidate cache for
	hold, err := h.repo.GetHoldByID(holdID)
	if err != nil {
		if err.Error() == "hold not found" {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Error:   "not_found",
				Message: "Hold not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get hold details",
		})
		return
	}

	err = h.repo.ReleaseHold(holdID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to release hold",
		})
		return
	}

	// Invalidate seat-related caches since seats were released
	h.cache.InvalidateAvailableSeats(hold.EventID)
	h.cache.InvalidateAvailableSeatCount(hold.EventID)

	c.JSON(http.StatusOK, gin.H{"message": "Hold released successfully"})
}

// ConfirmHold handles confirming a seat hold (booking)
func (h *EventHandler) ConfirmHold(c *gin.Context) {
	holdIDStr := c.Param("holdId")
	holdID, err := uuid.Parse(holdIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid hold ID format",
		})
		return
	}

	// Get hold first to know which event to invalidate cache for
	hold, err := h.repo.GetHoldByID(holdID)
	if err != nil {
		if err.Error() == "hold not found" {
			c.JSON(http.StatusNotFound, model.ErrorResponse{
				Error:   "not_found",
				Message: "Hold not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get hold details",
		})
		return
	}

	err = h.repo.ConfirmHold(holdID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to confirm hold",
		})
		return
	}

	// Invalidate seat-related caches since seats were booked
	h.cache.InvalidateAvailableSeats(hold.EventID)
	h.cache.InvalidateAvailableSeatCount(hold.EventID)

	c.JSON(http.StatusOK, gin.H{"message": "Booking confirmed successfully"})
}

// HealthCheck handles health check endpoint
func (h *EventHandler) HealthCheck(c *gin.Context) {
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
		Service:   "event-service",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
