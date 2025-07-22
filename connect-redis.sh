#!/bin/bash

# Redis connection script for event-booking system
# Usage: ./connect-redis.sh

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Redis configuration from docker-compose.yml
REDIS_HOST="localhost"
REDIS_PORT="6379"
REDIS_PASSWORD=""  # No password set in docker-compose.yml
REDIS_DB="0"       # Default database

# Function to display usage and info
show_info() {
    echo -e "${BLUE}=== Event Booking Redis Connection Script ===${NC}"
    echo ""
    echo -e "${YELLOW}Redis Configuration:${NC}"
    echo "  Host: $REDIS_HOST"
    echo "  Port: $REDIS_PORT"
    echo "  Database: $REDIS_DB"
    echo "  Password: <not set>"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0              - Connect to Redis"
    echo "  $0 --help       - Show this help message"
    echo ""
    echo -e "${YELLOW}Common Redis CLI commands once connected:${NC}"
    echo "  PING                    - Test connection"
    echo "  INFO                    - Show Redis server info"
    echo "  KEYS *                  - List all keys (use carefully in production!)"
    echo "  TYPE key_name           - Show type of a key"
    echo "  GET key_name            - Get string value"
    echo "  HGETALL key_name        - Get hash values"
    echo "  LRANGE key_name 0 -1    - Get list values"
    echo "  TTL key_name            - Check key expiration"
    echo "  FLUSHDB                 - Clear current database (use with caution!)"
    echo "  SELECT db_number        - Switch database (0-15)"
    echo "  QUIT                    - Exit redis-cli"
    echo ""
    echo -e "${YELLOW}Cache-related commands (for your services):${NC}"
    echo "  KEYS event:*            -- Show event cache keys"
    echo "  KEYS booking:*          -- Show booking cache keys"
    echo "  GET event:123           -- Get cached event data"
    echo "  HGETALL booking:456     -- Get cached booking hash"
    echo ""
    echo -e "${YELLOW}Note:${NC} Make sure Docker Compose services are running before connecting."
    echo "Run: ${GREEN}docker-compose up${NC} or ${GREEN}./start-dev.sh${NC}"
}

# Function to check if redis-cli is installed
check_redis_cli() {
    if ! command -v redis-cli &> /dev/null; then
        echo -e "${RED}Error: redis-cli is not installed or not in PATH${NC}"
        echo "Please install Redis client tools to use this script."
        echo ""
        echo "Installation instructions:"
        echo "  macOS: brew install redis"
        echo "  Ubuntu/Debian: sudo apt-get install redis-tools"
        echo "  CentOS/RHEL: sudo yum install redis"
        exit 1
    fi
}

# Function to check if Redis service is running
check_service_running() {
    if ! nc -z $REDIS_HOST $REDIS_PORT 2>/dev/null; then
        echo -e "${RED}Warning: Redis service appears to be down${NC}"
        echo "Redis is not responding on $REDIS_HOST:$REDIS_PORT"
        echo ""
        echo "To start the services:"
        echo "  docker-compose up -d redis"
        echo "  # or start all services:"
        echo "  ./start-dev.sh"
        echo ""
        echo -e "${YELLOW}Attempting connection anyway...${NC}"
        echo ""
    else
        echo -e "${GREEN}Redis service is running on $REDIS_HOST:$REDIS_PORT${NC}"
    fi
}

# Function to connect to Redis
connect_to_redis() {
    echo -e "${GREEN}Connecting to Redis...${NC}"
    echo -e "${BLUE}Connection: $REDIS_HOST:$REDIS_PORT (database $REDIS_DB)${NC}"
    echo -e "${YELLOW}Type QUIT to exit redis-cli${NC}"
    echo ""
    
    # Connect to Redis
    if [ -n "$REDIS_PASSWORD" ]; then
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" -n "$REDIS_DB"
    else
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -n "$REDIS_DB"
    fi
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
        echo -e "${GREEN}Proceeding with Redis connection...${NC}"
        echo ""
    fi
    
    # Check if redis-cli is available
    check_redis_cli
    
    # Check if service is running
    check_service_running
    
    # Connect to Redis
    connect_to_redis
}

# Run the main function with all arguments
main "$@"
