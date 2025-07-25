# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the grad binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o grad ./cmd/grad

# Final stage
FROM debian:bookworm-slim

# Install ca-certificates and curl for HTTPS requests and health checks
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the grad binary from builder
COPY --from=builder /app/grad .

# Create non-root user
RUN useradd -r -s /bin/false appuser
USER appuser

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the grad service
CMD ["./grad"]