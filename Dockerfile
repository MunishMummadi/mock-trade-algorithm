# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o mock-trade-algorithm .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite tzdata

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/mock-trade-algorithm .

# Create data directory
RUN mkdir -p ./data

# Create config directory
RUN mkdir -p ./config

# Copy configuration template
COPY --from=builder /app/config/.env.example ./config/.env.example

# Set executable permissions
RUN chmod +x ./mock-trade-algorithm

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD pgrep mock-trade-algorithm || exit 1

# Run the application
CMD ["./mock-trade-algorithm"] 