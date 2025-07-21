#!/bin/bash

# Event Booking System - Development Stop Script
echo "🛑 Stopping Event Booking System Development Environment"
echo "=================================================="

# Stop all services
echo "⏹️  Stopping all services..."
docker-compose down

# Optional: Remove volumes (uncomment if you want to clear data)
# echo "🗑️  Removing volumes and data..."
# docker-compose down -v

# Optional: Remove images (uncomment if you want to clean up completely)
# echo "🧹 Removing built images..."
# docker-compose down --rmi all

echo "✅ Event Booking System stopped successfully!"
echo ""
echo "💡 To start again, run: ./start-dev.sh"
