FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache make git gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# We use the Makefile build target to ensure consistency
RUN make build

# Final stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/trader /app/trader

# Copy configuration
COPY config.yaml ./

# Prometheus metrics and HTTP/WebSocket API
EXPOSE 9090
EXPOSE 8080

# Set default env vars for RPCs to empty string if not provided
ENV ARBITRUM_RPC_HTTP="" \
    ARBITRUM_RPC_WS="" \
    BASE_RPC_HTTP="" \
    BASE_RPC_WS="" \
    POLYGON_RPC_HTTP="" \
    POLYGON_RPC_WS="" \
    PRIVATE_KEY=""

ENTRYPOINT ["/app/trader", "-config", "/app/config.yaml"]
