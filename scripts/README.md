# Scripts Directory

This directory contains load testing tools for the Event Booking System.

## Load Testing

### `load-test-rps.js`
k6 script for testing specific requests-per-second (RPS) rates against your event service.

**Usage:**
1. Edit the script to set your desired RPS:
   ```javascript
   const TARGET_RPS = 100;        // Change this value
   const TEST_DURATION = '60s';   // How long to run the test
   ```

2. Install k6:
   ```bash
   # macOS
   brew install k6
   
   # Linux
   curl -LO https://github.com/grafana/k6/releases/latest/download/k6-linux-amd64.tar.gz
   tar -xzf k6-linux-amd64.tar.gz
   sudo mv k6-*/k6 /usr/local/bin/
   ```

3. Start your services:
   ```bash
   cd ..
   ./start-dev.sh
   ```

4. Run the load test:
   ```bash
   k6 run load-test-rps.js
   ```

**What it tests:**
- GET `/api/events/:id` endpoint
- Maintains constant RPS rate
- Measures response times and error rates
- Provides detailed metrics

**Key metrics to watch:**
- `http_reqs`: Actual RPS achieved
- `http_req_duration`: Response time percentiles
- `http_req_failed`: Error rate percentage
- `success_rate`: Custom success metric

**Example output:**
```
✅ GOOD RESULTS:
http_reqs......................: 3000    50/s     ← Achieved target RPS
http_req_duration..............: avg=45ms p(95)=120ms  ← Fast responses
http_req_failed................: 0.12%   ← Low error rate
success_rate...................: 99.8%   ← High success rate
```

### `load-test-booking-flow.js` ⭐ NEW
k6 script for comprehensive booking flow load testing focusing on booking submissions and status checks.

**Usage:**
1. Make sure all services are running:
   ```bash
   cd ..
   ./start-dev.sh
   ```

2. Edit the script to set your desired RPS:
   ```javascript
   const CONFIG = {
     TARGET_RPS: 10,              // Booking submissions per second
     TEST_DURATION: '60s',        // How long to run the test
     SEATS_PER_EVENT: 100,        // Seats per event
     SETUP_EVENTS_COUNT: 5,       // Events to pre-create
   };
   ```

3. Run the test:
   ```bash
   k6 run load-test-booking-flow.js
   ```

**What it tests:**
- **Complete booking workflow** end-to-end
- **User authentication** (`POST /api/users/login`) and token management
- **Event creation** (`POST /api/events`) with seat inventory
- **Seat hold creation** (`POST /api/events/:id/hold`) with random seat selection
- **📈 Booking submission** (MAIN FOCUS) - `POST /api/booking`
- **📈 Booking status checking** (MAIN FOCUS) - `GET /api/booking/:id`
- **Realistic payment flow** with mock credit card details

**Key metrics to watch:**
- `booking_submissions`: Total booking attempts
- `successful_bookings`: Completed bookings  
- `booking_success_rate`: Success percentage (target: >80%)
- **`booking_response_time`**: Booking submission response times (MAIN FOCUS)
- **`status_check_time`**: Status check response times (MAIN FOCUS)
- `hold_creation_time`: Seat hold creation times
- `authentication_time`: User login times

**Test Flow per Virtual User:**
1. Login with test user (gets JWT token)
2. Create seat hold for 1-3 random seats on random event
3. Submit booking with payment info (returns booking ID)
4. Check booking status immediately (validates processing status)
5. Track success/failure rates and response times

**Expected Results:**
- Booking submissions should be <5s (95th percentile)
- Status checks should be <1s (95th percentile) 
- Success rate should be >80%
- HTTP error rate should be <10%

## Configuration

### RPS Test (`load-test-rps.js`)
Edit these values:
- `TARGET_RPS`: Requests per second to test
- `TEST_DURATION`: How long to run the test
- `BASE_URL`: Service URL (default: http://localhost:8082)
- `EVENT_IDS`: Array of event IDs to test with

### Booking Flow Test (`load-test-booking-flow.js`)
Edit these values:
- `TARGET_RPS`: Booking submissions per second
- `TEST_DURATION`: How long to run the test
- `SEATS_PER_EVENT`: Number of seats per event
- `SETUP_EVENTS_COUNT`: Number of events to create
- Service URLs (if different from defaults)
