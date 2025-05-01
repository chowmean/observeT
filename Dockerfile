# Build stage
FROM golang:1.22-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o observeT main.go config.go

# Runtime stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from the build stage
COPY --from=builder /app/observeT /app/

# Copy configuration file
COPY config.yaml /app/

# Create directory structure for grafana and prometheus configs
COPY grafana/ /app/grafana/
COPY prometheus/ /app/prometheus/

# Expose Prometheus metrics port
EXPOSE 9095

# Set entrypoint
ENTRYPOINT ["/app/observeT"]

# Default command (can be overridden)
CMD []