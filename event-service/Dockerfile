# Build stage
FROM golang:1.22.5-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY event-service/go.mod event-service/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY event-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8082

# Run the application
CMD ["./main"]
