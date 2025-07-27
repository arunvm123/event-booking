import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// ===== CONFIGURATION =====
// Define CONFIG first before using it
const CONFIG = {
  TARGET_RPS: 10,              // Booking submissions per second
  TEST_DURATION: '60s',        // How long to run the test
  SEATS_PER_EVENT: 1000000,       // Seats per event (matching test script)
  SETUP_EVENTS_COUNT: 1,       // Events to pre-create
};

// Service URLs
const USER_SERVICE = 'http://localhost:8081';
const EVENT_SERVICE = 'http://localhost:8082';
const BOOKING_SERVICE = 'http://localhost:8083';

// Custom metrics
const bookingSubmissions = new Counter('booking_submissions');
const successfulBookings = new Counter('successful_bookings');
const failedBookings = new Counter('failed_bookings');
const bookingSuccessRate = new Rate('booking_success_rate');
const bookingResponseTime = new Trend('booking_response_time');
const statusCheckTime = new Trend('status_check_time');
const holdCreationTime = new Trend('hold_creation_time');
const authenticationTime = new Trend('authentication_time');
const finalConfirmationTime = new Trend('final_confirmation_time');
const seatsAlreadyHeld = new Counter('seats_already_held');
const bookingTimeouts = new Counter('booking_timeouts');

// Test configuration for constant RPS
// Now CONFIG is defined, so we can use it here
export const options = {
  scenarios: {
    booking_flow: {
      executor: 'constant-arrival-rate',
      rate: CONFIG.TARGET_RPS,
      timeUnit: '1s',
      duration: CONFIG.TEST_DURATION,
      preAllocatedVUs: 20,        // Increased from 1 to handle concurrent requests
      maxVUs: 50,                 // Increased from 1 to allow scaling
    },
  },
  
  thresholds: {
    'booking_success_rate': ['rate>0.80'],         // 80% success rate
    'booking_response_time': ['p(95)<5000'],       // 95% under 5 seconds
    'status_check_time': ['p(95)<1000'],           // 95% under 1 second
    'final_confirmation_time': ['p(95)<30000'],    // 95% under 30 seconds for full processing
    'http_req_duration': ['p(95)<6000'],           // Overall 95% under 6 seconds
    'http_req_failed': ['rate<0.10'],              // Less than 10% failed requests
  },
};

// Global test data
let testUsers = [];
let testEvents = [];
let preallocatedSeats = []; // Pre-allocated seat combinations to avoid overlaps
let jwtTokens = new Map();

export function setup() {
  console.log(`🚀 Starting Booking Flow Load Test`);
  console.log(`📊 Target: ${CONFIG.TARGET_RPS} booking submissions per second for ${CONFIG.TEST_DURATION}`);
  console.log(`🎯 Focus: POST /api/booking and GET /api/booking/:id/status`);
  
  // Health checks
  console.log('🏥 Performing health checks...');
  const services = [
    { name: 'User Service', url: USER_SERVICE },
    { name: 'Event Service', url: EVENT_SERVICE },
    { name: 'Booking Service', url: BOOKING_SERVICE },
  ];
  
  for (const service of services) {
    const healthResponse = http.get(`${service.url}/health`);
    if (healthResponse.status !== 200) {
      console.error(`❌ ${service.name} health check failed!`);
      throw new Error(`${service.name} not available`);
    } else {
      console.log(`✅ ${service.name} is healthy`);
    }
  }
  
  // Create test users
  console.log(`👥 Creating ${CONFIG.SETUP_EVENTS_COUNT * 2} test users...`);
  for (let i = 0; i < CONFIG.SETUP_EVENTS_COUNT * 2; i++) {
    const timestamp = Date.now();
    const randomId = Math.random().toString(36).substring(7);
    const email = `testuser.${timestamp}.${randomId}@loadtest.com`;
    const password = 'password123';
    
    // Register user
    const registerResponse = http.post(`${USER_SERVICE}/api/users/register`, JSON.stringify({
      first_name: 'LoadTest',
      last_name: `User${i}`,
      email: email,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (registerResponse.status === 201 || registerResponse.status === 200) {
      // Login to get JWT token
      const loginResponse = http.post(`${USER_SERVICE}/api/users/login`, JSON.stringify({
        email: email,
        password: password,
      }), {
        headers: { 'Content-Type': 'application/json' },
      });
      
      if (loginResponse.status === 200) {
        const loginData = JSON.parse(loginResponse.body);
        const token = loginData.access_token;
        
        testUsers.push({
          email: email,
          password: password,
          token: token,
        });
        
        console.log(`✅ Created user ${i + 1}: ${email.substring(0, 20)}...`);
      } else {
        console.log(`❌ Failed to login user ${i + 1}: ${email}`);
      }
    } else {
      console.log(`❌ Failed to register user ${i + 1}: ${email}`);
    }
  }
  
  if (testUsers.length === 0) {
    throw new Error('No test users created successfully');
  }
  
  // Create test events
  console.log(`🎪 Creating ${CONFIG.SETUP_EVENTS_COUNT} test events...`);
  for (let i = 0; i < CONFIG.SETUP_EVENTS_COUNT; i++) {
    const userToken = testUsers[i % testUsers.length].token;
    const eventDate = new Date();
    eventDate.setDate(eventDate.getDate() + 30); // 30 days from now
    
    const eventResponse = http.post(`${EVENT_SERVICE}/api/events`, JSON.stringify({
      name: `Load Test Concert ${i + 1}`,
      venue: `Test Venue ${i + 1}`,
      city: 'Load Test City',
      category: 'Music',
      event_date: eventDate.toISOString(),
      total_seats: CONFIG.SEATS_PER_EVENT,
      price_per_seat: 99.99,
      description: `Load test event ${i + 1}`,
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${userToken}`,
      },
    });
    
    if (eventResponse.status === 201 || eventResponse.status === 200) {
      const eventData = JSON.parse(eventResponse.body);
      testEvents.push({
        event_id: eventData.event_id,
        name: eventData.name,
        venue: eventData.venue,
        total_seats: CONFIG.SEATS_PER_EVENT,
      });
      
      console.log(`✅ Created event ${i + 1}: ${eventData.event_id}`);
    } else {
      console.log(`❌ Failed to create event ${i + 1}: Status ${eventResponse.status}`);
    }
  }
  
  if (testEvents.length === 0) {
    throw new Error('No test events created successfully');
  }
  
  // Pre-allocate seat combinations to avoid overlaps during load testing
  console.log('🎟️ Pre-allocating seat combinations to avoid overlaps...');
  const seatCombinations = generateSeatCombinations();
  console.log(`✅ Generated ${seatCombinations.length} unique seat combinations`);
  
  console.log(`🎯 Setup complete: ${testUsers.length} users, ${testEvents.length} events, ${seatCombinations.length} seat combinations`);
  console.log('🚀 Starting load test...\n');
  
  return {
    users: testUsers,
    events: testEvents,
    seatCombinations: seatCombinations,
    startTime: Date.now(),
  };
}

export default function (data) {
  // Select random user and event
  const user = data.users[Math.floor(Math.random() * data.users.length)];
  const event = data.events[Math.floor(Math.random() * data.events.length)];
  
  // Better seat selection - use modulo to cycle through combinations
  const vuId = __VU;
  const iterationId = __ITER;
  
  // Calculate a unique index that cycles through available combinations
  const seatComboIndex = (vuId * 1000 + iterationId) % data.seatCombinations.length;
  const seatNumbers = data.seatCombinations[seatComboIndex];
  
  // Debug: Log the seat combination being used
  console.log(`🎟️ VU ${vuId}, Iteration ${iterationId}: Using seats [${seatNumbers.join(', ')}] for event ${event.event_id}`);
  
  // Step 1: Create seat hold
  const holdStartTime = Date.now();
  const holdResponse = http.post(`${EVENT_SERVICE}/api/events/${event.event_id}/hold`, JSON.stringify({
    seat_numbers: seatNumbers,
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${user.token}`,
    },
    tags: { name: 'create_hold' },
  });
  
  holdCreationTime.add(Date.now() - holdStartTime);
  
  // Check hold creation
  const holdSuccess = check(holdResponse, {
    'hold creation status is 200/201': (r) => r.status === 200 || r.status === 201,
    'hold response has hold_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hold_id !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  if (!holdSuccess) {
    console.log(`❌ Hold creation failed: Status ${holdResponse.status}, Body: ${holdResponse.body}`);
    try {
      const errorBody = JSON.parse(holdResponse.body);
      if (errorBody.error && errorBody.error.includes('already held')) {
        seatsAlreadyHeld.add(1);
      }
    } catch {
      // Error parsing error response
    }
    return;
  }
  
  const holdData = JSON.parse(holdResponse.body);
  const holdId = holdData.hold_id;
  
  // Step 2: Submit booking (MAIN FOCUS)
  const bookingStartTime = Date.now();
  const totalAmount = seatNumbers.length * 99.99;
  
  const bookingResponse = http.post(`${BOOKING_SERVICE}/api/booking`, JSON.stringify({
    hold_id: holdId,
    payment_info: {
      amount: totalAmount,
      payment_method: 'credit_card',
    },
  }), {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${user.token}`,
    },
    tags: { name: 'submit_booking' },
  });
  
  const bookingDuration = Date.now() - bookingStartTime;
  bookingResponseTime.add(bookingDuration);
  bookingSubmissions.add(1);
  
  // Check booking submission
  const bookingSubmitSuccess = check(bookingResponse, {
    'booking submission status is 202': (r) => r.status === 202,
    'booking response has booking_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.booking_id !== undefined;
      } catch {
        return false;
      }
    },
    'booking response has status URL': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status_url !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  if (!bookingSubmitSuccess) {
    console.log(`❌ Booking submission failed: Status ${bookingResponse.status}, Body: ${bookingResponse.body}`);
    failedBookings.add(1);
    bookingSuccessRate.add(0);
    return;
  }
  
  const bookingData = JSON.parse(bookingResponse.body);
  const bookingId = bookingData.booking_id;
  const pollingStartTime = Date.now();
  
  // Step 3: Poll for final status
  let finalStatus = 'processing';
  let pollAttempts = 0;
  const maxPolls = 15; // 30 seconds max (15 * 2 second intervals)
  let lastStatusResponse;
  
  while ((finalStatus === 'processing' || finalStatus === 'pending') && pollAttempts < maxPolls) {
    // Wait before polling (except first attempt)
    if (pollAttempts > 0) {
      sleep(2);
    }
    
    const statusResponse = http.get(`${BOOKING_SERVICE}/api/booking/${bookingId}/status`, {
      headers: {
        'Authorization': `Bearer ${user.token}`,
      },
      tags: { name: 'poll_status' },
    });
    
    statusCheckTime.add(statusResponse.timings.duration);
    
    if (statusResponse.status === 200) {
      try {
        const statusData = JSON.parse(statusResponse.body);
        finalStatus = statusData.status;
        lastStatusResponse = statusData;
        
        // Log status transitions
        if (pollAttempts === 0 || finalStatus !== 'processing') {
          console.log(`📊 Booking ${bookingId}: ${finalStatus} (attempt ${pollAttempts + 1})`);
        }
      } catch (e) {
        console.error(`Failed to parse status response: ${statusResponse.body}`);
      }
    } else {
      console.error(`Status check failed: ${statusResponse.status}`);
    }
    
    pollAttempts++;
  }
  
  // Record final confirmation time
  const totalProcessingTime = Date.now() - pollingStartTime;
  finalConfirmationTime.add(totalProcessingTime);
  
  // Track final outcomes
  if (finalStatus === 'confirmed') {
    successfulBookings.add(1);
    bookingSuccessRate.add(1);
    console.log(`✅ Booking ${bookingId} confirmed in ${totalProcessingTime}ms`);
  } else if (finalStatus === 'failed') {
    failedBookings.add(1);
    bookingSuccessRate.add(0);
    const errorMsg = lastStatusResponse?.error_message || 'Unknown error';
    console.log(`❌ Booking ${bookingId} failed: ${errorMsg}`);
  } else {
    // Still processing after max attempts
    bookingTimeouts.add(1);
    bookingSuccessRate.add(0);
    console.log(`⏱️ Booking ${bookingId} timed out after ${maxPolls * 2} seconds`);
  }
}

// Generate non-overlapping seat combinations for load testing
function generateSeatCombinations() {
  const combinations = [];
  
  // Calculate estimated number of booking requests during test
  const testDurationSeconds = parseInt(CONFIG.TEST_DURATION.replace('s', ''));
  const estimatedRequests = CONFIG.TARGET_RPS * testDurationSeconds;
  const maxVUs = 50; // From options
  const bufferMultiplier = 2.0; // Increased buffer for VU distribution
  const targetCombinations = Math.floor(estimatedRequests * bufferMultiplier);
  
  console.log(`🧮 Generating seat combinations:`);
  console.log(`   • Test duration: ${testDurationSeconds}s`);
  console.log(`   • Target RPS: ${CONFIG.TARGET_RPS}`);
  console.log(`   • Estimated requests: ${estimatedRequests}`);
  console.log(`   • Target combinations (with ${bufferMultiplier}x buffer): ${targetCombinations}`);
  
  // Calculate how many rows we have (1000 seats / 50 seats per row = 20 rows)
  const totalRows = Math.ceil(CONFIG.SEATS_PER_EVENT / 50);
  console.log(`   • Total rows available: ${totalRows} (A to ${String.fromCharCode(64 + totalRows)})`);
  
  // Generate combinations using multiple strategies to ensure uniqueness
  let generatedCount = 0;
  
  // Strategy 1: Sequential pairs (A1-A2, A3-A4, etc.)
  for (let row = 0; row < totalRows && generatedCount < targetCombinations; row++) {
    const rowLetter = String.fromCharCode(65 + row); // A, B, C...
    
    for (let seatStart = 1; seatStart <= 49 && generatedCount < targetCombinations; seatStart += 2) {
      combinations.push([
        `${rowLetter}${seatStart}`,
        `${rowLetter}${seatStart + 1}`
      ]);
      generatedCount++;
    }
  }
  
  // Strategy 2: If we need more, use non-adjacent seats
  if (generatedCount < targetCombinations) {
    for (let row = 0; row < totalRows && generatedCount < targetCombinations; row++) {
      const rowLetter = String.fromCharCode(65 + row);
      
      for (let gap = 2; gap <= 25 && generatedCount < targetCombinations; gap++) {
        for (let seatStart = 1; seatStart + gap <= 50 && generatedCount < targetCombinations; seatStart++) {
          combinations.push([
            `${rowLetter}${seatStart}`,
            `${rowLetter}${seatStart + gap}`
          ]);
          generatedCount++;
        }
      }
    }
  }
  
  console.log(`   • Generated ${combinations.length} unique seat combinations`);
  console.log(`   • Combinations per VU: ~${Math.floor(combinations.length / maxVUs)}`);
  console.log(`   • First combination: [${combinations[0].join(', ')}]`);
  console.log(`   • Last combination: [${combinations[combinations.length - 1].join(', ')}]`);
  
  // Shuffle to distribute evenly across VUs
  for (let i = combinations.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [combinations[i], combinations[j]] = [combinations[j], combinations[i]];
  }
  
  return combinations;
}

export function teardown(data) {
  const testDurationMs = Date.now() - data.startTime;
  const testDurationSeconds = testDurationMs / 1000;
  
  console.log('');
  console.log('📋 === BOOKING FLOW LOAD TEST SUMMARY ===');
  console.log(`🎯 Target RPS: ${CONFIG.TARGET_RPS}`);
  console.log(`⏱️  Test Duration: ${testDurationSeconds.toFixed(2)} seconds`);
  console.log(`👥 Test Users: ${data.users.length}`);
  console.log(`🎪 Test Events: ${data.events.length}`);
  console.log(`🎟️ Seat Combinations Used: ${data.seatCombinations.length}`);
  console.log('');
  console.log('📈 Key Metrics to Review:');
  console.log('   • booking_submissions: Total booking attempts');
  console.log('   • successful_bookings: Completed bookings');
  console.log('   • booking_success_rate: Success percentage (target: >80%)');
  console.log('   • booking_response_time: Booking submission response times (target: p95 <5s)');
  console.log('   • status_check_time: Status check response times (target: p95 <1s)');
  console.log('   • final_confirmation_time: Time to reach final status (target: p95 <30s)');
  console.log('   • booking_timeouts: Bookings that didn\'t complete in time');
  console.log('   • seats_already_held: Conflicts due to seat availability');
  console.log('');
  console.log('🎯 Expected Results:');
  console.log('   • Booking submissions should be <5s (95th percentile)');
  console.log('   • Status checks should be <1s (95th percentile)');
  console.log('   • Final confirmation should be <30s (95th percentile)');
  console.log('   • Success rate should be >80%');
  console.log('   • HTTP error rate should be <10%');
  console.log('');
  console.log('✅ Booking flow load test completed!');
}