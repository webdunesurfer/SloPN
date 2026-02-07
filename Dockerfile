# Stage 1: Build the Go binary
FROM golang:1.25-bullseye AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o slopn-server ./cmd/server/main.go

# Stage 2: Final lean image
FROM debian:bullseye-slim

# Install networking tools required for TUN and NAT
RUN apt-get update && apt-get install -y \
    iptables \
    iproute2 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/slopn-server .

# Expose the default QUIC/UDP port
EXPOSE 4242/udp

# Run the server
# Default flags can be overridden by CMD or environment variables in docker-compose
ENTRYPOINT ["./slopn-server"]
