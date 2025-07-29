import http from 'k6/http';
import { check } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// ===== CONFIGURATION =====
const TARGET_RPS = 12;        // Change this to your desired requests per second
const TEST_DURATION = '60s';   // How long to run the test
const BASE_URL = 'http://localhost:8082';

// Custom metrics
const successRate = new Rate('success_rate');
const errorRate = new Rate('error_rate');
const requestCount = new Counter('requests_total');
const responseTimeP95 = new Trend('response_time_p95');

// Test configuration for constant RPS
export const options = {
  scenarios: {
    constant_rps: {
      executor: 'constant-arrival-rate',
      rate: TARGET_RPS,           // Target RPS
      timeUnit: '1s',             // Per second
      duration: TEST_DURATION,    // Test duration
      preAllocatedVUs: 10,        // Initial VUs
      maxVUs: 200,                // Maximum VUs if needed
    },
  },
  
  thresholds: {
    'success_rate': ['rate>0.95'],           // 95% success rate
    'error_rate': ['rate<0.05'],             // Less than 5% error rate
    'http_req_duration': ['p(95)<1000'],     // 95% under 1 second
    'http_req_failed': ['rate<0.05'],        // Less than 5% failed requests
  },
};

// Test data - modify these based on your actual event IDs
const EVENT_IDS = ['5893dcc4-c123-49e1-a576-abb8ce95f160', '1d1da5e9-5ceb-4e68-87a7-bf357012e81f', '818810a7-f917-40d5-adf1-1266376612c6'];

export function setup() {
  console.log(`🚀 Starting RPS Load Test`);
  console.log(`📊 Target: ${TARGET_RPS} requests per second for ${TEST_DURATION}`);
  console.log(`🎯 Endpoint: ${BASE_URL}/api/events/:id`);
  
  // Health check
  const healthResponse = http.get(`${BASE_URL}/health`);
  if (healthResponse.status !== 200) {
    console.error('❌ Service health check failed! Make sure the event service is running.');
    throw new Error('Service not available');
  } else {
    console.log('✅ Event service is healthy');
  }
  
  return { startTime: Date.now() };
}

export default function (data) {
  // Select a random event ID
  const eventId = EVENT_IDS[Math.floor(Math.random() * EVENT_IDS.length)];
  const url = `${BASE_URL}/api/events/${eventId}`;
  
  // Make the request
  const response = http.get(url, {
    tags: { 
      name: 'get_event',
      event_id: eventId
    },
  });
  
  // Count total requests
  requestCount.add(1);
  
  // Record response time
  responseTimeP95.add(response.timings.duration);
  
  // Check if request was successful
  const isSuccess = check(response, {
    'status is 200 or 404': (r) => r.status === 200 || r.status === 404,
    'response time < 2000ms': (r) => r.timings.duration < 2000,
    'response has body': (r) => r.body && r.body.length > 0,
  });
  
  // Track success/error rates
  if (isSuccess) {
    successRate.add(1);
    errorRate.add(0);
  } else {
    successRate.add(0);
    errorRate.add(1);
    console.log(`❌ Request failed for event ${eventId}: Status ${response.status}, Duration: ${response.timings.duration}ms`);
  }
}

export function teardown(data) {
  const testDurationMs = Date.now() - data.startTime;
  const testDurationSeconds = testDurationMs / 1000;
  
  console.log('');
  console.log('📋 === LOAD TEST SUMMARY ===');
  console.log(`🎯 Target RPS: ${TARGET_RPS}`);
  console.log(`⏱️  Test Duration: ${testDurationSeconds.toFixed(2)} seconds`);
  console.log(`📊 Expected Total Requests: ~${TARGET_RPS * (testDurationSeconds / 1000) * parseInt(TEST_DURATION)}`);
  console.log('');
  console.log('📈 Key Metrics to Review:');
  console.log('   • http_req_duration: Response time statistics');
  console.log('   • http_reqs: Actual requests per second achieved');
  console.log('   • success_rate: Percentage of successful requests');
  console.log('   • error_rate: Percentage of failed requests');
  console.log('   • dropped_iterations: Requests that couldn\'t be made (if any)');
  console.log('');
  console.log('✅ Test completed successfully!');
}
