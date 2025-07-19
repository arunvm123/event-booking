# User Service

This is the User Service for the Event Booking System. It handles user registration, authentication, and JWT token management.

## Features

- User registration with email validation
- User authentication with JWT tokens
- Password hashing with bcrypt
- PostgreSQL database integration using GORM
- Health check endpoint
- CORS support
- Request logging middleware

## API Endpoints

### Public Endpoints (No Authentication Required)

#### 1. User Registration
```http
POST /api/users/register
Content-Type: application/json

{
  "email": "john.doe@example.com",
  "password": "securePassword123",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response (201 Created):**
```json
{
  "user_id": "uuid-here",
  "email": "john.doe@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "created_at": "2025-07-19T10:30:00Z"
}
```

#### 2. User Login
```http
POST /api/users/login
Content-Type: application/json

{
  "email": "john.doe@example.com",
  "password": "securePassword123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "jwt-token-here",
  "expires_in": 3600,
  "user": {
    "user_id": "uuid-here",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe"
  }
}
```

#### 3. Health Check
```http
GET /health
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "service": "user-service",
  "timestamp": "2025-07-19T12:00:00Z"
}
```

## Configuration

The service uses **cleanenv** for configuration management and supports both YAML files and environment variables.

### Environment Variables

- `PORT`: Server port (default: `8081`)
- `DB_USER`: Database username (default: `postgres`)
- `DB_PASSWORD`: Database password (default: `password`)
- `DB_NAME`: Database name (default: `eventbooking`)
- `DB_HOST`: Database host (default: `localhost`)
- `DB_PORT`: Database port (default: `5432`)
- `DB_SSL_MODE`: Database SSL mode (default: `disable`)
- `JWT_SECRET`: Secret key for JWT token signing (default: `your-secret-key-change-in-production`)

### Configuration File

You can also use a YAML configuration file. See `config.yaml` for an example:

```yaml
port: "8081"

database:
  user: "postgres"
  password: "password"
  database_name: "eventbooking"
  host: "localhost"
  port: "5432"
  ssl_mode: "disable"

jwt_secret: "your-secret-key-change-in-production"
```

## Running the Service

### Prerequisites

1. Go 1.22.5 or later
2. PostgreSQL database

### Local Development

1. Start PostgreSQL database
2. Set environment variables (optional)
3. Run the service:

```bash
go run .
```

The service will start on `:8081` by default.

### Database

The service uses GORM auto-migration, so the `users` table will be created automatically when the service starts.

## Dependencies

- **Gin**: HTTP web framework
- **GORM**: ORM for PostgreSQL
- **JWT**: JSON Web Token implementation
- **bcrypt**: Password hashing
- **UUID**: UUID generation

## Architecture Notes

- The service follows clean architecture principles with separate layers for handlers, repository, and models
- JWT tokens expire after 1 hour
- Passwords are hashed using bcrypt with default cost
- Database connections are managed by GORM
- CORS is enabled for cross-origin requests
- Request logging is implemented for monitoring
