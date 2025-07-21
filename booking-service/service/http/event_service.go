package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/arunvm123/eventbooking/booking-service/config"
	"github.com/arunvm123/eventbooking/booking-service/service"
	"github.com/golang-jwt/jwt/v5"
)

// JWTServiceInterface defines the interface for JWT operations
type JWTServiceInterface interface {
	GenerateServiceToken(userID, userEmail string) (string, error)
}

// JWTService handles JWT operations for service-to-service communication
type JWTService struct {
	secretKey string
}

func NewJWTService(secretKey string) *JWTService {
	return &JWTService{secretKey: secretKey}
}

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateServiceToken generates a JWT token for service-to-service communication
func (j *JWTService) GenerateServiceToken(userID, userEmail string) (string, error) {
	// Create claims for service-to-service communication with actual user context
	claims := Claims{
		UserID: userID,
		Email:  userEmail,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "booking-service",
			Subject:   "service-auth",
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response
	tokenString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

type HTTPEventService struct {
	baseURL    string
	httpClient *http.Client
	jwtService JWTServiceInterface
}

func NewHTTPEventService(baseURL, jwtSecret string) *HTTPEventService {
	return &HTTPEventService{
		baseURL:    baseURL,
		jwtService: NewJWTService(jwtSecret),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewHTTPEventServiceWithConfig creates a new HTTP event service with connection pooling
func NewHTTPEventServiceWithConfig(cfg *config.EventService, jwtSecret string) *HTTPEventService {
	// Create HTTP transport with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     time.Duration(cfg.IdleConnTimeout) * time.Second,
		DisableKeepAlives:   false, // Enable keep-alive for connection reuse
		ForceAttemptHTTP2:   true,  // Enable HTTP/2 for better multiplexing
	}

	return &HTTPEventService{
		baseURL:    cfg.BaseURL,
		jwtService: NewJWTService(jwtSecret),
		httpClient: &http.Client{
			Timeout:   time.Duration(cfg.RequestTimeout) * time.Second,
			Transport: transport,
		},
	}
}

// GetHoldDetails retrieves hold information from the event service
func (s *HTTPEventService) GetHoldDetails(holdID, userID, userEmail string) (*service.HoldDetails, error) {
	url := fmt.Sprintf("%s/api/events/holds/%s", s.baseURL, holdID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Generate JWT token for service-to-service authentication with user context
	token, err := s.jwtService.GenerateServiceToken(userID, userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service token: %w", err)
	}

	// Add internal service authentication header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("hold not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("event service error (status %d): %s", resp.StatusCode, string(body))
	}

	var holdDetails service.HoldDetails
	if err := json.NewDecoder(resp.Body).Decode(&holdDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &holdDetails, nil
}

// ConfirmHold confirms a hold (converts it to booking) in the event service
func (s *HTTPEventService) ConfirmHold(holdID, userID, userEmail string) error {
	url := fmt.Sprintf("%s/api/events/holds/%s/confirm", s.baseURL, holdID)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Generate JWT token for service-to-service authentication with user context
	token, err := s.jwtService.GenerateServiceToken(userID, userEmail)
	if err != nil {
		return fmt.Errorf("failed to generate service token: %w", err)
	}

	// Add internal service authentication header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("hold not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("event service error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReleaseHold releases a hold in the event service
func (s *HTTPEventService) ReleaseHold(holdID, userID, userEmail string) error {
	url := fmt.Sprintf("%s/api/events/holds/%s", s.baseURL, holdID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Generate JWT token for service-to-service authentication with user context
	token, err := s.jwtService.GenerateServiceToken(userID, userEmail)
	if err != nil {
		return fmt.Errorf("failed to generate service token: %w", err)
	}

	// Add internal service authentication header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("hold not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("event service error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
