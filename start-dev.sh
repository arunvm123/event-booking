#!/bin/bash

# Event Booking System - Development Startup Script
echo "🚀 Starting Event Booking System Development Environment"
echo "=================================================="

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install Docker Compose."
    exit 1
fi

# Parse command line arguments
FORCE_REBUILD=false
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --rebuild|-r) FORCE_REBUILD=true ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Clean up any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose down --remove-orphans

# Force rebuild if requested
if [ "$FORCE_REBUILD" = true ]; then
    echo "🔨 Force rebuilding all services..."
    docker-compose build --no-cache
else
    echo "🏗️  Building services (with cache)..."
    docker-compose build
fi

# Remove any dangling images (optional - uncomment if needed)
# echo "🗑️  Removing dangling images..."
# docker image prune -f

# Start infrastructure services first
echo "📦 Starting infrastructure services (PostgreSQL, Redis, Kafka)..."
docker-compose up -d postgres redis zookeeper kafka

# Wait for infrastructure to be ready
echo "⏳ Waiting for infrastructure services to be healthy..."
sleep 15

# Check health of critical services
echo "🏥 Checking service health..."
docker-compose ps

# Start application services with build flag
echo "🎯 Starting application services..."
docker-compose up -d --build user-service event-service booking-service-api notification-service-api

# Wait a bit for API services to start
sleep 10

# Start worker services with build flag
echo "⚙️  Starting worker services..."
docker-compose up -d --build booking-service-worker notification-service-worker

echo ""
echo "✅ Event Booking System Started Successfully!"
echo "=================================================="
echo ""
echo "🌐 Service URLs:"
echo "   User Service:         http://localhost:8081"
echo "   Event Service:        http://localhost:8082" 
echo "   Booking Service:      http://localhost:8083"
echo "   Notification Service: http://localhost:8084"
echo ""
echo "📊 Infrastructure URLs:"
echo "   PostgreSQL:           localhost:5433"
echo "   Redis:                localhost:6379"
echo "   Kafka:                localhost:9092"
echo "   Zookeeper:            localhost:2181"
echo ""
echo "🔍 Useful Commands:"
echo "   View logs:            docker-compose logs -f [service-name]"
echo "   Stop all services:    docker-compose down"
echo "   Restart service:      docker-compose restart [service-name]"
echo "   View service status:  docker-compose ps"
echo "   Force rebuild:        ./start-dev.sh --rebuild"
echo ""
echo "📝 API Testing:"
echo "   Health checks:        curl http://localhost:808[1-4]/health"
echo "   User registration:    curl -X POST http://localhost:8081/api/auth/register"
echo "   Event creation:       curl -X POST http://localhost:8082/api/events"
echo "   Booking creation:     curl -X POST http://localhost:8083/api/booking"
echo ""
echo "🎉 Happy coding! The system is ready for development and testing."