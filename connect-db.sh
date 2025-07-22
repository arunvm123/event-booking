#!/bin/bash

# Database connection script for event-booking system
# Usage: ./connect-db.sh

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database configuration from docker-compose.yml
DB_HOST="localhost"
DB_PORT="5433"
DB_NAME="eventbooking"
DB_USER="postgres"
DB_PASSWORD="postgres"

# Function to display usage and info
show_info() {
    echo -e "${BLUE}=== Event Booking Database Connection Script ===${NC}"
    echo ""
    echo -e "${YELLOW}Database Configuration:${NC}"
    echo "  Host: $DB_HOST"
    echo "  Port: $DB_PORT"
    echo "  Database: $DB_NAME"
    echo "  User: $DB_USER"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0              - Connect to the main eventbooking database"
    echo "  $0 --help       - Show this help message"
    echo ""
    echo -e "${YELLOW}Common psql commands once connected:${NC}"
    echo "  \\l              - List all databases"
    echo "  \\dt             - List tables in current database"
    echo "  \\d table_name   - Describe table structure"
    echo "  \\q              - Quit psql"
    echo ""
    echo -e "${YELLOW}Service-specific queries:${NC}"
    echo "  SELECT * FROM users;           -- User service data"
    echo "  SELECT * FROM events;          -- Event service data"
    echo "  SELECT * FROM bookings;        -- Booking service data"
    echo "  SELECT * FROM notifications;   -- Notification service data"
    echo ""
    echo -e "${YELLOW}Note:${NC} Make sure Docker Compose services are running before connecting."
    echo "Run: ${GREEN}docker-compose up${NC} or ${GREEN}./start-dev.sh${NC}"
}

# Function to check if psql is installed
check_psql() {
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}Error: psql is not installed or not in PATH${NC}"
        echo "Please install PostgreSQL client tools to use this script."
        echo ""
        echo "Installation instructions:"
        echo "  macOS: brew install postgresql"
        echo "  Ubuntu/Debian: sudo apt-get install postgresql-client"
        echo "  CentOS/RHEL: sudo yum install postgresql"
        exit 1
    fi
}

# Function to check if database service is running
check_service_running() {
    if ! nc -z $DB_HOST $DB_PORT 2>/dev/null; then
        echo -e "${RED}Warning: Database service appears to be down${NC}"
        echo "PostgreSQL is not responding on $DB_HOST:$DB_PORT"
        echo ""
        echo "To start the services:"
        echo "  docker-compose up -d postgres"
        echo "  # or start all services:"
        echo "  ./start-dev.sh"
        echo ""
        echo -e "${YELLOW}Attempting connection anyway...${NC}"
        echo ""
    else
        echo -e "${GREEN}Database service is running on $DB_HOST:$DB_PORT${NC}"
    fi
}

# Function to connect to database
connect_to_db() {
    local connection_string="host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME sslmode=disable"
    
    echo -e "${GREEN}Connecting to eventbooking database...${NC}"
    echo -e "${BLUE}Connection: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME${NC}"
    echo -e "${YELLOW}Type \\q to quit psql${NC}"
    echo ""
    
    # Set PGPASSWORD environment variable to avoid password prompt
    export PGPASSWORD="$DB_PASSWORD"
    
    # Connect to the database
    psql "$connection_string"
}

# Main script logic
main() {
    # Handle help flag
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        show_info
        exit 0
    fi
    
    # Show info if no arguments
    if [ $# -eq 0 ]; then
        show_info
        echo -e "${GREEN}Proceeding with database connection...${NC}"
        echo ""
    fi
    
    # Check if psql is available
    check_psql
    
    # Check if service is running
    check_service_running
    
    # Connect to the database
    connect_to_db
}

# Run the main function with all arguments
main "$@"
