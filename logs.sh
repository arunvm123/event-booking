#!/bin/bash

# Event Booking System - Logs Viewer Script
echo "📋 Event Booking System Logs"
echo "=================================================="

if [ -z "$1" ]; then
    echo "Usage: ./logs.sh [service-name] [optional: number of lines]"
    echo ""
    echo "Available services:"
    echo "   postgres"
    echo "   redis" 
    echo "   zookeeper"
    echo "   kafka"
    echo "   user-service"
    echo "   event-service"
    echo "   booking-service-api"
    echo "   booking-service-worker"
    echo "   notification-service-api"
    echo "   notification-service-worker"
    echo ""
    echo "Examples:"
    echo "   ./logs.sh booking-service-worker"
    echo "   ./logs.sh booking-service-api 100"
    echo "   ./logs.sh all (for all services)"
    exit 1
fi

SERVICE=$1
LINES=${2:-50}

if [ "$SERVICE" = "all" ]; then
    echo "📊 Showing logs for all services..."
    docker-compose logs --tail=$LINES -f
else
    echo "📊 Showing logs for $SERVICE (last $LINES lines)..."
    docker-compose logs --tail=$LINES -f $SERVICE
fi
