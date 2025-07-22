#!/bin/bash

# Kafka connection script for event-booking system using kcat
# Usage: ./connect-kafka.sh [command] [options]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Kafka configuration from docker-compose.yml
KAFKA_BROKER="localhost:9092"

# Function to display usage and info
show_info() {
    echo -e "${BLUE}=== Event Booking Kafka Connection Script ===${NC}"
    echo ""
    echo -e "${YELLOW}Kafka Configuration:${NC}"
    echo "  Broker: $KAFKA_BROKER"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0                      - Show this help and list topics"
    echo "  $0 list                 - List all topics"
    echo "  $0 consume <topic>      - Consume messages from a topic"
    echo "  $0 produce <topic>      - Produce messages to a topic"
    echo "  $0 info <topic>         - Show topic information"
    echo "  $0 metadata             - Show cluster metadata"
    echo "  $0 --help               - Show this help message"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 list                           # List all topics"
    echo "  $0 consume booking.events         # Consume from booking events topic"
    echo "  $0 produce notification.events    # Produce to notification events topic"
    echo "  $0 info booking.events            # Show booking events topic info"
    echo ""
    echo -e "${YELLOW}Common Kafka topics for your event-booking system:${NC}"
    echo "  booking.events          -- Booking-related events"
    echo "  notification.events     -- Notification events"
    echo "  user.events            -- User-related events"
    echo "  event.events           -- Event management events"
    echo ""
    echo -e "${YELLOW}Interactive consume commands:${NC}"
    echo "  Ctrl+C                 -- Stop consuming"
    echo "  -o beginning           -- Start from beginning of topic"
    echo "  -o end                 -- Start from end of topic"
    echo ""
    echo -e "${YELLOW}Note:${NC} Make sure Docker Compose services are running before connecting."
    echo "Run: ${GREEN}docker-compose up${NC} or ${GREEN}./start-dev.sh${NC}"
}

# Function to check if kcat is installed
check_kcat() {
    if ! command -v kcat &> /dev/null; then
        echo -e "${RED}Error: kcat is not installed or not in PATH${NC}"
        echo "Please install kcat (formerly kafkacat) to use this script."
        echo ""
        echo "Installation instructions:"
        echo "  macOS: brew install kcat"
        echo "  Ubuntu/Debian: sudo apt-get install kafkacat"
        echo "  CentOS/RHEL: sudo yum install kafkacat"
        echo ""
        echo "Alternative: Use Docker to run kcat:"
        echo "  alias kcat='docker run --rm -it --network host confluentinc/cp-kafkacat:latest'"
        exit 1
    fi
}

# Function to check if Kafka service is running
check_kafka_running() {
    if ! nc -z localhost 9092 2>/dev/null; then
        echo -e "${RED}Warning: Kafka service appears to be down${NC}"
        echo "Kafka is not responding on $KAFKA_BROKER"
        echo ""
        echo "To start the services:"
        echo "  docker-compose up -d kafka"
        echo "  # or start all services:"
        echo "  ./start-dev.sh"
        echo ""
        echo -e "${YELLOW}Attempting connection anyway...${NC}"
        echo ""
    else
        echo -e "${GREEN}Kafka service is running on $KAFKA_BROKER${NC}"
    fi
}

# Function to list topics
list_topics() {
    echo -e "${GREEN}Listing Kafka topics...${NC}"
    echo ""
    kcat -b "$KAFKA_BROKER" -L | grep -E "topic \".*\"" | sed 's/.*topic "\(.*\)" .*/\1/' | sort
}

# Function to show cluster metadata
show_metadata() {
    echo -e "${GREEN}Kafka cluster metadata:${NC}"
    echo ""
    kcat -b "$KAFKA_BROKER" -L
}

# Function to show topic information
show_topic_info() {
    local topic=$1
    if [ -z "$topic" ]; then
        echo -e "${RED}Error: Topic name is required${NC}"
        echo "Usage: $0 info <topic_name>"
        exit 1
    fi
    
    echo -e "${GREEN}Topic information for: ${CYAN}$topic${NC}"
    echo ""
    kcat -b "$KAFKA_BROKER" -t "$topic" -L
}

# Function to consume messages
consume_messages() {
    local topic=$1
    if [ -z "$topic" ]; then
        echo -e "${RED}Error: Topic name is required${NC}"
        echo "Usage: $0 consume <topic_name>"
        exit 1
    fi
    
    echo -e "${GREEN}Consuming messages from topic: ${CYAN}$topic${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop consuming${NC}"
    echo ""
    
    # Start consuming from the beginning with formatted output
    kcat -b "$KAFKA_BROKER" -t "$topic" -C -o beginning -f '\nKey: %k\nValue: %s\nPartition: %p\nOffset: %o\nTimestamp: %T\n%R\n'
}

# Function to produce messages
produce_messages() {
    local topic=$1
    if [ -z "$topic" ]; then
        echo -e "${RED}Error: Topic name is required${NC}"
        echo "Usage: $0 produce <topic_name>"
        exit 1
    fi
    
    echo -e "${GREEN}Producing messages to topic: ${CYAN}$topic${NC}"
    echo -e "${YELLOW}Type your messages and press Enter. Press Ctrl+C to stop.${NC}"
    echo -e "${YELLOW}You can also pipe messages: echo 'message' | $0 produce $topic${NC}"
    echo ""
    
    # Start producing messages
    kcat -b "$KAFKA_BROKER" -t "$topic" -P
}

# Main script logic
main() {
    local command=$1
    local topic=$2
    
    # Handle help flag
    if [[ "$command" == "--help" || "$command" == "-h" ]]; then
        show_info
        exit 0
    fi
    
    # Check if kcat is available
    check_kcat
    
    # Check if Kafka is running
    check_kafka_running
    
    case "$command" in
        "list")
            list_topics
            ;;
        "consume")
            consume_messages "$topic"
            ;;
        "produce")
            produce_messages "$topic"
            ;;
        "info")
            show_topic_info "$topic"
            ;;
        "metadata")
            show_metadata
            ;;
        "")
            # No command provided, show info and list topics
            show_info
            echo ""
            echo -e "${GREEN}Available topics:${NC}"
            list_topics
            ;;
        *)
            echo -e "${RED}Error: Unknown command '$command'${NC}"
            echo ""
            show_info
            exit 1
            ;;
    esac
}

# Run the main function with all arguments
main "$@"
