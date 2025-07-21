package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arunvm123/eventbooking/booking-service/model"
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

// Cache key generator
func (r *RedisCacheRepository) bookingStatusKey(bookingID string) string {
	return fmt.Sprintf("booking_status:%s", bookingID)
}

// GetBookingStatus retrieves booking status update from cache
func (r *RedisCacheRepository) GetBookingStatus(bookingID string) (*model.BookingStatusUpdate, error) {
	key := r.bookingStatusKey(bookingID)
	statusData, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var status model.BookingStatusUpdate
	if err := json.Unmarshal([]byte(statusData), &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SetBookingStatus stores booking status update in cache
func (r *RedisCacheRepository) SetBookingStatus(bookingID string, status *model.BookingStatusUpdate, ttl time.Duration) error {
	key := r.bookingStatusKey(bookingID)
	statusData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, statusData, ttl).Err()
}

// InvalidateBookingStatus removes booking status from cache
func (r *RedisCacheRepository) InvalidateBookingStatus(bookingID string) error {
	key := r.bookingStatusKey(bookingID)
	return r.client.Del(r.ctx, key).Err()
}

// Ping checks if Redis is healthy
func (r *RedisCacheRepository) Ping() error {
	return r.client.Ping(r.ctx).Err()
}
