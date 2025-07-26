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

## Configuration

Edit these values in `load-test-rps.js`:
- `TARGET_RPS`: Requests per second to test
- `TEST_DURATION`: How long to run the test
- `BASE_URL`: Service URL (default: http://localhost:8082)
- `EVENT_IDS`: Array of event IDs to test with
