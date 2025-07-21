#!/bin/bash

# Event Booking System - Successful Booking Test Script
echo "🧪 Testing Successful Booking Flow"
echo "=================================="

# Base URLs
USER_SERVICE="http://localhost:8081"
EVENT_SERVICE="http://localhost:8082"
BOOKING_SERVICE="http://localhost:8083"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if service is up
check_service() {
    local service_name=$1
    local service_url=$2
    
    echo -n "Checking $service_name... "
    if curl -s "$service_url/health" > /dev/null; then
        echo -e "${GREEN}✓ Online${NC}"
        return 0
    else
        echo -e "${RED}✗ Offline${NC}"
        return 1
    fi
}

# Check all services
echo "🏥 Health Checks:"
check_service "User Service" "$USER_SERVICE" || exit 1
check_service "Event Service" "$EVENT_SERVICE" || exit 1
check_service "Booking Service" "$BOOKING_SERVICE" || exit 1
echo ""

# Generate unique email for each test run
TIMESTAMP=$(date +%s)
USER_EMAIL="testuser.${TIMESTAMP}@example.com"

# Step 1: Create User
echo -e "${BLUE}📝 Step 1: Creating User${NC}"
USER_RESPONSE=$(curl -s -X POST "$USER_SERVICE/api/users/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"first_name\": \"John\",
    \"last_name\": \"Doe\", 
    \"email\": \"$USER_EMAIL\",
    \"password\": \"password123\"
  }")

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ User created successfully${NC}"
    echo "Response: $USER_RESPONSE"
    
    # Check if registration was successful
    if echo "$USER_RESPONSE" | grep -q '"error"'; then
        echo -e "${RED}✗ User registration failed${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Failed to create user${NC}"
    exit 1
fi
echo ""

# Step 1.5: Login to get JWT token
echo -e "${BLUE}🔐 Step 1.5: Logging in to get JWT token${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$USER_SERVICE/api/users/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$USER_EMAIL\",
    \"password\": \"password123\"
  }")

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ Login successful${NC}"
    echo "Response: $LOGIN_RESPONSE"
    
    # Extract JWT token
    JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    if [[ -n "$JWT_TOKEN" ]]; then
        echo "JWT Token: ${JWT_TOKEN:0:20}..."
    else
        echo -e "${RED}✗ Failed to extract JWT token${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Failed to login${NC}"
    exit 1
fi
echo ""

# Step 2: Create Event
echo -e "${BLUE}🎪 Step 2: Creating Event${NC}"
EVENT_RESPONSE=$(curl -s -X POST "$EVENT_SERVICE/api/events" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{
    "name": "Concert 2024",
    "venue": "Madison Square Garden",
    "city": "New York",
    "category": "Music",
    "event_date": "2024-12-31T20:00:00Z",
    "total_seats": 1000,
    "price_per_seat": 99.99,
    "description": "Amazing New Year Concert"
  }')

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ Event created successfully${NC}"
    echo "Response: $EVENT_RESPONSE"
    
    # Extract Event ID
    EVENT_ID=$(echo "$EVENT_RESPONSE" | grep -o '"event_id":"[^"]*"' | cut -d'"' -f4)
    if [[ -n "$EVENT_ID" ]]; then
        echo "Event ID: $EVENT_ID"
    else
        echo -e "${RED}✗ Failed to extract Event ID${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Failed to create event${NC}"
    exit 1
fi
echo ""

# Step 3: Create Seat Hold
echo -e "${BLUE}🎟️ Step 3: Creating Seat Hold${NC}"
HOLD_RESPONSE=$(curl -s -X POST "$EVENT_SERVICE/api/events/$EVENT_ID/hold" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d '{
    "seat_numbers": ["A1", "A2"]
  }')

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ Seat hold created successfully${NC}"
    echo "Response: $HOLD_RESPONSE"
    
    # Extract Hold ID
    HOLD_ID=$(echo "$HOLD_RESPONSE" | grep -o '"hold_id":"[^"]*"' | cut -d'"' -f4)
    if [[ -n "$HOLD_ID" ]]; then
        echo "Hold ID: $HOLD_ID"
    else
        echo -e "${RED}✗ Failed to extract Hold ID${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Failed to create seat hold${NC}"
    exit 1
fi
echo ""

# Step 4: Create Booking
echo -e "${BLUE}💳 Step 4: Creating Booking${NC}"
BOOKING_RESPONSE=$(curl -s -X POST "$BOOKING_SERVICE/api/booking" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -d "{
    \"hold_id\": \"$HOLD_ID\",
    \"payment_info\": {
      \"amount\": 199.98,
      \"payment_method\": \"credit_card\"
    }
  }")

if [[ $? -eq 0 ]]; then
    echo -e "${GREEN}✓ Booking submitted successfully${NC}"
    echo "Response: $BOOKING_RESPONSE"
    
    # Extract Booking ID
    BOOKING_ID=$(echo "$BOOKING_RESPONSE" | grep -o '"booking_id":"[^"]*"' | cut -d'"' -f4)
    if [[ -n "$BOOKING_ID" ]]; then
        echo "Booking ID: $BOOKING_ID"
    else
        echo -e "${RED}✗ Failed to extract Booking ID${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Failed to create booking${NC}"
    exit 1
fi
echo ""

# Step 5: Monitor Booking Status
echo -e "${BLUE}� Step 5: Monitoring Booking Status${NC}"
echo "Waiting for booking to be processed by worker pool..."

for i in {1..10}; do
    sleep 3
    STATUS_RESPONSE=$(curl -s -X GET "$BOOKING_SERVICE/api/booking/$BOOKING_ID/status" \
      -H "Authorization: Bearer $JWT_TOKEN")
    
    if [[ $? -eq 0 ]]; then
        echo "[$i] Status check: $STATUS_RESPONSE"
        
        # Check if booking is confirmed
        if echo "$STATUS_RESPONSE" | grep -q '"status":"confirmed"'; then
            echo -e "${GREEN}🎉 SUCCESS! Booking confirmed!${NC}"
            echo ""
            echo "📊 Final Status: $STATUS_RESPONSE"
            exit 0
        elif echo "$STATUS_RESPONSE" | grep -q '"status":"failed"'; then
            echo -e "${RED}❌ Booking failed${NC}"
            echo "Final Status: $STATUS_RESPONSE"
            exit 1
        fi
    else
        echo "[$i] Failed to check status"
    fi
done

echo -e "${RED}⏰ Timeout waiting for booking confirmation${NC}"
exit 1
