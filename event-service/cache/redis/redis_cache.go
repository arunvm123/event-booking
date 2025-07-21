package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arunvm123/eventbooking/event-service/model"
	"github.com/redis/go-redis/v9"
)

type RedisCacheRepository struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCacheRepository(redisURL, password string, db int) (*RedisCacheRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCacheRepository{
		client: client,
		ctx:    ctx,
	}, nil
}

// Cache key generators
func (r *RedisCacheRepository) availableSeatsKey(eventID string) string {
	return fmt.Sprintf("event:%s:seats:available", eventID)
}

func (r *RedisCacheRepository) availableSeatCountKey(eventID string) string {
	return fmt.Sprintf("event:%s:seats:count", eventID)
}

func (r *RedisCacheRepository) eventKey(eventID string) string {
	return fmt.Sprintf("event:%s:details", eventID)
}

func (r *RedisCacheRepository) eventListKey(filterKey string) string {
	return fmt.Sprintf("events:list:%s", filterKey)
}

// Seat availability caching
func (r *RedisCacheRepository) GetAvailableSeats(eventID string) ([]string, error) {
	key := r.availableSeatsKey(eventID)
	seats, err := r.client.SMembers(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}
	return seats, nil
}

func (r *RedisCacheRepository) SetAvailableSeats(eventID string, seats []string, ttl time.Duration) error {
	key := r.availableSeatsKey(eventID)

	// Clear existing set
	if err := r.client.Del(r.ctx, key).Err(); err != nil {
		return err
	}

	// Add all seats to set
	if len(seats) > 0 {
		seatInterfaces := make([]interface{}, len(seats))
		for i, seat := range seats {
			seatInterfaces[i] = seat
		}
		if err := r.client.SAdd(r.ctx, key, seatInterfaces...).Err(); err != nil {
			return err
		}
	}

	// Set expiration
	return r.client.Expire(r.ctx, key, ttl).Err()
}

func (r *RedisCacheRepository) InvalidateAvailableSeats(eventID string) error {
	key := r.availableSeatsKey(eventID)
	return r.client.Del(r.ctx, key).Err()
}

// Seat count caching
func (r *RedisCacheRepository) GetAvailableSeatCount(eventID string) (int, error) {
	key := r.availableSeatCountKey(eventID)
	countStr, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, nil // Cache miss
		}
		return -1, err
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return -1, err
	}

	return count, nil
}

func (r *RedisCacheRepository) SetAvailableSeatCount(eventID string, count int, ttl time.Duration) error {
	key := r.availableSeatCountKey(eventID)
	return r.client.Set(r.ctx, key, count, ttl).Err()
}

func (r *RedisCacheRepository) InvalidateAvailableSeatCount(eventID string) error {
	key := r.availableSeatCountKey(eventID)
	return r.client.Del(r.ctx, key).Err()
}

// Event details caching
func (r *RedisCacheRepository) GetEvent(eventID string) (*model.Event, error) {
	key := r.eventKey(eventID)
	eventData, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var event model.Event
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, err
	}

	return &event, nil
}

func (r *RedisCacheRepository) SetEvent(eventID string, event *model.Event, ttl time.Duration) error {
	key := r.eventKey(eventID)
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, eventData, ttl).Err()
}

func (r *RedisCacheRepository) InvalidateEvent(eventID string) error {
	key := r.eventKey(eventID)
	return r.client.Del(r.ctx, key).Err()
}

// Event list caching
func (r *RedisCacheRepository) GetEventList(filterKey string) (*model.EventListResponse, error) {
	key := r.eventListKey(filterKey)
	listData, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var eventList model.EventListResponse
	if err := json.Unmarshal([]byte(listData), &eventList); err != nil {
		return nil, err
	}

	return &eventList, nil
}

func (r *RedisCacheRepository) SetEventList(filterKey string, response *model.EventListResponse, ttl time.Duration) error {
	key := r.eventListKey(filterKey)
	listData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, listData, ttl).Err()
}

func (r *RedisCacheRepository) InvalidateEventList(pattern string) error {
	searchPattern := fmt.Sprintf("events:list:%s", pattern)
	keys, err := r.client.Keys(r.ctx, searchPattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}

	return nil
}

// Health check
func (r *RedisCacheRepository) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

// Cache invalidation for event-related data
func (r *RedisCacheRepository) InvalidateEventRelatedCache(eventID string) error {
	// Invalidate all event-related cache entries
	keys := []string{
		r.eventKey(eventID),
		r.availableSeatsKey(eventID),
		r.availableSeatCountKey(eventID),
	}

	if err := r.client.Del(r.ctx, keys...).Err(); err != nil {
		return err
	}

	// Invalidate event list caches (they might contain this event)
	return r.InvalidateEventList("*")
}

// Utility method to generate cache key for filtered event lists
func GenerateFilterKey(filter model.EventFilter) string {
	var parts []string

	if filter.City != "" {
		parts = append(parts, fmt.Sprintf("city:%s", filter.City))
	}
	if filter.Category != "" {
		parts = append(parts, fmt.Sprintf("cat:%s", filter.Category))
	}
	if filter.Name != "" {
		parts = append(parts, fmt.Sprintf("name:%s", filter.Name))
	}
	if filter.DateFrom != nil {
		parts = append(parts, fmt.Sprintf("from:%s", filter.DateFrom.Format("2006-01-02")))
	}
	if filter.DateTo != nil {
		parts = append(parts, fmt.Sprintf("to:%s", filter.DateTo.Format("2006-01-02")))
	}

	parts = append(parts, fmt.Sprintf("limit:%d", filter.Limit))
	parts = append(parts, fmt.Sprintf("offset:%d", filter.Offset))

	if len(parts) == 0 {
		return "all"
	}

	return strings.Join(parts, ":")
}
