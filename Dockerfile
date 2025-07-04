# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o emailserver cmd/emailserver/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/emailserver .

# Copy default config
COPY --from=builder /app/config/example.yaml /app/config/config.yaml

# Create necessary directories
RUN mkdir -p /data /config /logs

# Expose ports
EXPOSE 587 8080

# Use non-root user
RUN adduser -D -u 1000 emailserver
USER emailserver

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./emailserver", "-config", "/config/config.yaml"]