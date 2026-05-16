# ==========================================
# STAGE 1: Build the Go Binary
# ==========================================
FROM golang:1.23-alpine AS builder

# Install git and build-base for any potential C dependencies (though CGO is disabled)
RUN apk add --no-cache git

WORKDIR /app

# 1. Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy the entire source (required for modular service architecture)
COPY . .

# 3. Compile the WASM Game Engine (Ensures client/server rule parity)
RUN GOOS=js GOARCH=wasm go build -o Public/main.wasm main.go

# 4. Build the optimized, static Server binary
# -ldflags="-w -s" removes debug information to reduce size
# CGO_ENABLED=0 ensures the binary is statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server-bin server.go

# ==========================================
# STAGE 2: Minimal Runtime Image
# ==========================================
# Pin to a specific minor version for stability
FROM alpine:3.18

# Install CA certificates (Required for HTTPS calls to blockchain Indexers/RPCs)
# and tzdata for correct timestamp logging
RUN apk --no-cache add ca-certificates tzdata curl

# Create a non-root user and group for better security
RUN addgroup -S arena && adduser -S arenabot -G arena

WORKDIR /app

# Create a data directory for persistent volumes (Render support)
RUN mkdir -p /app/data && chown -R arenabot:arena /app/data

# Set Environment Variable for the data directory
ENV DATA_DIR=/app/data

# Copy the entrypoint script and make it executable
COPY --chown=arenabot:arena entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Copy only the compiled binary from the builder stage
COPY --from=builder --chown=arenabot:arena /app/server-bin .

# Copy the frontend assets (HTML, JS, WASM, and Audio/Images)
# WARNING: Public/main.wasm must be built and included before this image is deployed.
# If main.wasm is missing, the browser frontend will not load correctly.
COPY --chown=arenabot:arena Public/ ./Public/

# Switch to the non-root user
USER arenabot

# Render typically uses a dynamic PORT, but our server defaults to 8082
EXPOSE 8082

# Health check to ensure the API is responding
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:${PORT:-8082}/api/health || exit 1

# Run the server
CMD ["./server-bin"]