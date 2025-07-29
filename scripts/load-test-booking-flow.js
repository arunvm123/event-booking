import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// ===== CONFIGURATION =====
// Define CONFIG first before using it
const CONFIG = {
  TARGET_RPS: 10,              // Booking submissions per second
  TEST_DURATION: '60s',        // How long to run the test
  SEATS_PER_EVENT: 50000,      // 50k seats per event (plenty for no conflicts)
  SETUP_USERS_COUNT: 20,       // 20 users, each mapped to their own event
  SETUP_EVENTS_COUNT: 20,      // 20 events, one per user
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
  console.log(`👥 Creating ${CONFIG.SETUP_USERS_COUNT} test users...`);
  for (let i = 0; i < CONFIG.SETUP_USERS_COUNT; i++) {
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
          id: i,  // User ID for mapping to event
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
  
  // Create test events - one per user
  console.log(`🎪 Creating ${CONFIG.SETUP_EVENTS_COUNT} test events (one per user)...`);
  for (let i = 0; i < CONFIG.SETUP_EVENTS_COUNT; i++) {
    const userToken = testUsers[i].token;  // Use the corresponding user's token
    const eventDate = new Date();
    eventDate.setDate(eventDate.getDate() + 30); // 30 days from now
    
    const eventResponse = http.post(`${EVENT_SERVICE}/api/events`, JSON.stringify({
      name: `Load Test Concert ${i + 1} - User ${i + 1} Exclusive`,
      venue: `Test Venue ${i + 1}`,
      city: 'Load Test City',
      category: 'Music',
      event_date: eventDate.toISOString(),
      total_seats: CONFIG.SEATS_PER_EVENT,
      price_per_seat: 99.99,
      description: `Load test event ${i + 1} - dedicated for user ${i + 1}`,
    }), {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${userToken}`,
      },
    });
    
    if (eventResponse.status === 201 || eventResponse.status === 200) {
      const eventData = JSON.parse(eventResponse.body);
      testEvents.push({
        id: i,  // Event ID for mapping to user
        event_id: eventData.event_id,
        name: eventData.name,
        venue: eventData.venue,
        total_seats: CONFIG.SEATS_PER_EVENT,
        assigned_user_id: i,  // Map this event to user i
      });
      
      console.log(`✅ Created event ${i + 1}: ${eventData.event_id} (assigned to user ${i + 1})`);
    } else {
      console.log(`❌ Failed to create event ${i + 1}: Status ${eventResponse.status}`);
    }
  }
  
  if (testEvents.length === 0) {
    throw new Error('No test events created successfully');
  }
  
  // No need for complex seat allocation - with 50k seats per event, we can use simple sequential allocation
  console.log('🎟️ Seat allocation strategy: Sequential allocation (no conflicts with 50k seats per event)');
  
  console.log(`\n🎯 Setup complete:`);
  console.log(`   • ${testUsers.length} users created`);
  console.log(`   • ${testEvents.length} events created (one per user)`);
  console.log(`   • ${CONFIG.SEATS_PER_EVENT.toLocaleString()} seats per event`);
  console.log(`   • User-Event mapping: Each user is assigned to their corresponding event`);
  console.log('🚀 Starting load test...\n');
  
  return {
    users: testUsers,
    events: testEvents,
    userEventMap: createUserEventMap(testUsers, testEvents),
    startTime: Date.now(),
  };
}

// Helper function to create user-event mapping
function createUserEventMap(users, events) {
  const map = {};
  users.forEach((user, index) => {
    if (events[index]) {
      map[user.id] = events[index];
    }
  });
  return map;
}

export default function (data) {
  // Use round-robin to select user based on VU and iteration
  const vuId = __VU;
  const iterationId = __ITER;
  const userIndex = (vuId - 1 + iterationId) % data.users.length;
  
  const user = data.users[userIndex];
  const event = data.userEventMap[user.id]; // Get the event assigned to this user
  
  if (!event) {
    console.error(`❌ No event found for user ${user.id}`);
    return;
  }
  
  // Generate unique sequential seats for this booking
  // With 50k seats, we can safely allocate seats without conflicts
  const uniqueOffset = (vuId * 1000) + (iterationId * 2); // Ensure uniqueness across VUs
  const seatStartNumber = uniqueOffset + 1; // Each booking gets 2 seats
  
  // Simple seat naming: Row A-Z, then AA-AZ, etc. (UPDATED: 500 seats per row)
  const rowNumber = Math.floor((seatStartNumber - 1) / 500); // 500 seats per row to match updated DB
  
  // Generate row name using EXACT same logic as Go code
  function generateRowName(index) {
    let result = '';
    while (true) {
      result = String.fromCharCode(65 + (index % 26)) + result;
      index = Math.floor(index / 26);
      if (index === 0) break;
      index--; // Adjust for the fact that there's no "zero" letter
    }
    return result;
  }
  
  const rowLetter = generateRowName(rowNumber);
  
  const seatInRow = ((seatStartNumber - 1) % 500) + 1; // Seats 1-500 per row
  
  // Make sure we don't exceed row capacity
  const seatNumbers = seatInRow <= 499 ? [
    `${rowLetter}${seatInRow}`,
    `${rowLetter}${seatInRow + 1}`
  ] : [
    `${rowLetter}${500}`, // Last seat in current row
    `A1` // First seat in row A (simple fallback)
  ];
  
  // Validate seat numbers before attempting to book
  const maxSeatNumber = Math.max(...seatNumbers.map(seat => {
    const match = seat.match(/[A-Z]+(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }));
  if (maxSeatNumber > 500) {
    console.log(`⚠️ Invalid seat numbers [${seatNumbers.join(', ')}] - max seat per row is 500, got ${maxSeatNumber}`);
    return;
  }
  
  console.log(`🎟️ User ${user.id} (VU${vuId}:${iterationId}) booking seats [${seatNumbers.join(', ')}] for event ${event.event_id}`);
  
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
    console.log(`❌ Hold creation failed for user ${user.id} (VU${vuId}:${iterationId}): Status ${holdResponse.status}`);
    console.log(`   Requested seats: [${seatNumbers.join(', ')}]`);
    console.log(`   Event ID: ${event.event_id}`);
    console.log(`   Response: ${holdResponse.body}`);
    
    // Try to get available seats to debug
    const availableSeatsResponse = http.get(`${EVENT_SERVICE}/api/events/${event.event_id}`, {
      headers: { 'Authorization': `Bearer ${user.token}` },
    });
    
    if (availableSeatsResponse.status === 200) {
      const eventData = JSON.parse(availableSeatsResponse.body);
      console.log(`   Available seats in event: ${eventData.available_seats}`);
      if (eventData.available_seat_numbers && eventData.available_seat_numbers.length > 0) {
        console.log(`   First 10 available: [${eventData.available_seat_numbers.slice(0, 10).join(', ')}]`);
      }
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
    console.log(`❌ Booking submission failed for user ${user.id}: Status ${bookingResponse.status}, Body: ${bookingResponse.body}`);
    failedBookings.add(1);
    bookingSuccessRate.add(0);
    return;
  }
  
  const bookingData = JSON.parse(bookingResponse.body);
  const createdBookingId = bookingData.booking_id;  // Changed variable name to avoid redeclaration
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
    
    const statusResponse = http.get(`${BOOKING_SERVICE}/api/booking/${createdBookingId}/status`, {
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
          console.log(`📊 Booking ${createdBookingId}: ${finalStatus} (attempt ${pollAttempts + 1})`);
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
    console.log(`✅ Booking ${createdBookingId} confirmed in ${totalProcessingTime}ms`);
  } else if (finalStatus === 'failed') {
    failedBookings.add(1);
    bookingSuccessRate.add(0);
    const errorMsg = lastStatusResponse?.error_message || 'Unknown error';
    console.log(`❌ Booking ${createdBookingId} failed: ${errorMsg}`);
  } else {
    // Still processing after max attempts
    bookingTimeouts.add(1);
    bookingSuccessRate.add(0);
    console.log(`⏱️ Booking ${createdBookingId} timed out after ${maxPolls * 2} seconds`);
  }
}

// No longer need complex seat generation function
// With 50k seats per event and user-event mapping, conflicts are virtually impossible

export function teardown(data) {
  const testDurationMs = Date.now() - data.startTime;
  const testDurationSeconds = testDurationMs / 1000;
  
  console.log('');
  console.log('📋 === BOOKING FLOW LOAD TEST SUMMARY ===');
  console.log(`🎯 Target RPS: ${CONFIG.TARGET_RPS}`);
  console.log(`⏱️  Test Duration: ${testDurationSeconds.toFixed(2)} seconds`);
  console.log(`👥 Test Users: ${data.users.length}`);
  console.log(`🎪 Test Events: ${data.events.length} (one per user)`);
  console.log(`🎟️ Seats per Event: ${CONFIG.SEATS_PER_EVENT.toLocaleString()}`);
  console.log('');
  console.log('📊 User-Event Mapping:');
  data.users.forEach((user, index) => {
    const event = data.events[index];
    if (event) {
      console.log(`   • User ${user.id} → Event ${event.event_id.substring(0, 8)}...`);
    }
  });
  console.log('');
  console.log('📈 Key Metrics to Review:');
  console.log('   • booking_submissions: Total booking attempts');
  console.log('   • successful_bookings: Completed bookings');
  console.log('   • booking_success_rate: Success percentage (target: >80%)');
  console.log('   • booking_response_time: Booking submission response times (target: p95 <5s)');
  console.log('   • status_check_time: Status check response times (target: p95 <1s)');
  console.log('   • final_confirmation_time: Time to reach final status (target: p95 <30s)');
  console.log('   • booking_timeouts: Bookings that didn\'t complete in time');
  console.log('');
  console.log('🎯 Expected Results:');
  console.log('   • NO seat conflicts (each user has their own event with 50k seats)');
  console.log('   • Booking submissions should be <5s (95th percentile)');
  console.log('   • Status checks should be <1s (95th percentile)');
  console.log('   • Final confirmation should be <30s (95th percentile)');
  console.log('   • Success rate should be >95% (higher due to no conflicts)');
  console.log('   • HTTP error rate should be <5%');
  console.log('');
  console.log('✅ Booking flow load test completed!');
}
