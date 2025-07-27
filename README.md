# Event Booking System

A scalable, microservices-based event booking platform built with Go, designed to handle high-traffic event ticketing scenarios with real-time seat management, asynchronous processing, and comprehensive notification systems.

##  Key Features

### High-Performance Booking System
- **Real-time seat management** with Redis caching
- **Asynchronous booking processing** using worker pools
- **Seat holding mechanism** with automatic expiration
- **Concurrent booking handling** with race condition prevention

### Microservices Architecture
- **Service isolation** with clear boundaries
- **Independent scaling** per service
- **Fault tolerance** with graceful degradation
- **Distributed caching** for optimal performance

### Notification System
- **Event-driven notifications** via Kafka
- **Email confirmations** for bookings
- **Real-time status updates** via Server-Sent Events (SSE)
- **Retry mechanisms** for failed notifications

## 🏗️ System Architecture

The system consists of four core microservices:

### User Service (Port 8081)
- User registration and authentication
- JWT token management
- User profile management
- Password hashing with bcrypt

### Event Service (Port 8082)
- Event creation and management
- Seat inventory management
- Seat holding with expiration
- Redis caching for performance

### Booking Service (Port 8083)
- Asynchronous booking processing
- Payment integration
- Worker pool architecture
- Real-time status updates via SSE

### Notification Service (Port 8084)
- Email notifications
- Booking confirmations
- Event reminders
- Kafka-based message processing

### Infrastructure Components
- **PostgreSQL**: Primary data storage
- **Redis**: Caching and session management
- **Kafka**: Event streaming and async messaging
- **Docker**: Containerization and orchestration

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.19+ (for development)

### Running the Application

1. **Start all services:**
   ```bash
   ./start-dev.sh
   ```

2. **Check service health:**
   ```bash
   curl http://localhost:8081/health  # User Service
   curl http://localhost:8082/health  # Event Service
   curl http://localhost:8083/health  # Booking Service
   curl http://localhost:8084/health  # Notification Service
   ```

3. **Stop all services:**
   ```bash
   ./stop-dev.sh
   ```

## 🧪 Testing the System

### 1. Test Successful Booking Flow
```bash
./test-successful-booking.sh
```
**This script demonstrates:**
- User registration and authentication
- Event creation with seat inventory
- Seat holding and booking process
- Real-time notification delivery
- End-to-end workflow validation

### 2. Test Failure Scenarios
```bash
./test-failed-booking.sh
```
**This script validates:**
- Invalid seat number handling
- Already booked seat detection
- Invalid hold ID rejection
- Non-existent event validation
- Worker pool resilience under load

### 3. Monitor System Logs
```bash
./logs.sh [service-name]  # Optional: specify user, event, booking, or notification
```

### 4. Load Testing with k6
Test your service's performance and capacity:

```bash
# Install k6 (if not already installed)
brew install k6  # macOS

# Run RPS load test (edit TARGET_RPS in the script first)
k6 run scripts/load-test-rps.js
```

**Available Scripts:**
- `scripts/load-test-rps.js` - k6 script for RPS (requests per second) load testing
- `scripts/README.md` - Load testing documentation and usage guide

### 5. Database & Infrastructure Debugging
```bash
# Connect to PostgreSQL database
./connect-db.sh

# Connect to Redis cache
./connect-redis.sh

# Connect to Kafka for message inspection
./connect-kafka.sh list
```

## 📚 API Documentation

### User Service (Port 8081)
- `POST /api/users/register` - User registration
- `POST /api/users/login` - User authentication
- `GET /api/users/profile` - Get user profile
- `PUT /api/users/profile` - Update user profile

### Event Service (Port 8082)
- `GET /api/events` - List events with filtering
- `POST /api/events` - Create new event
- `GET /api/events/{id}` - Get event details
- `POST /api/events/{id}/hold` - Create seat hold
- `DELETE /api/events/{id}/hold/{holdId}` - Release hold

### Booking Service (Port 8083)
- `POST /api/booking` - Submit booking with hold ID
- `GET /api/booking/{id}` - Get booking status
- `GET /api/booking/{id}/stream` - SSE status updates
- `GET /api/users/{userId}/bookings` - List user bookings

### Notification Service (Port 8084)
- `GET /health` - Service health check
- Internal Kafka consumer for processing notifications

## 🏛️ Data Flow

### Typical Booking Process
1. **User Registration**: Client → User Service → PostgreSQL
2. **Event Browsing**: Client → Event Service → Redis/PostgreSQL
3. **Seat Holding**: Client → Event Service → PostgreSQL + Redis Cache
4. **Booking Submission**: Client → Booking Service → Kafka Queue
5. **Async Processing**: Workers → Payment → Database Updates
6. **Notifications**: Booking Service → Kafka → Notification Service → Email

### Service Communication
- **Synchronous**: HTTP REST APIs for real-time operations
- **Asynchronous**: Kafka for event streaming and notifications
- **Caching**: Redis for session management and performance optimization

## 🛠️ Development

### Project Structure
```
event-booking/
├── user-service/           # User management and authentication
├── event-service/          # Event and seat management
├── booking-service/        # Booking processing and payments
├── notification-service/   # Email and SMS notifications
├── scripts/               # Load testing tools
│   ├── load-test-rps.js   # k6 RPS load testing script
│   └── README.md          # Load testing documentation
├── docker-compose.yml      # Infrastructure definition
├── start-dev.sh           # Development startup script
├── stop-dev.sh            # Cleanup script
├── logs.sh                # Log monitoring script
├── connect-db.sh          # PostgreSQL connection script
├── connect-redis.sh       # Redis connection script
├── connect-kafka.sh       # Kafka connection script
├── test-successful-booking.sh  # Integration tests
└── test-failed-booking.sh      # Error scenario tests
```

### Configuration
Each service uses YAML configuration files for:
- Database connections
- Redis settings
- Kafka brokers
- JWT secrets
- Email providers

## 📊 Monitoring & Observability

- **Health check endpoints** for all services
- **Structured logging** with request tracing
- **Error tracking** with detailed stack traces
- **Performance metrics** via application logs
- **Database connection monitoring**

## 🚀 Production Deployment

### Environment Setup
1. Configure production database connections
2. Set up Redis cluster for high availability
3. Deploy Kafka cluster with proper replication
4. Configure load balancers with health checks
5. Set up monitoring and alerting systems

### Performance Optimizations
- Database connection pooling
- Redis connection reuse
- Kafka producer batching
- HTTP keep-alive connections
- Caching strategies per service

---
