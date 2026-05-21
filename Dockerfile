# Stage 1: Build Environment (Go 1.24 + Node.js)
FROM golang:1.24-alpine3.20 AS builder

# Install system dependencies for build pipeline
RUN apk add --no-cache nodejs npm git

WORKDIR /app

# 1. Dependency Layer Optimization
COPY go.mod go.sum ./
RUN go mod download

COPY package*.json ./
RUN npm install

# 2. Source Ingestion
COPY . .

# 3. Execute Build Pipeline
# This handles wasm:init, wasm:build, sass:build, and server:build
RUN npm run build

# Stage 2: Hardened Production Runtime
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 4. Artifact Transfer
COPY --from=builder /app/server-bin .
COPY --from=builder /app/Public ./Public
COPY --from=builder /app/entrypoint.sh .
COPY --from=builder /app/networks.json .
# Optional: Copy season.json if it exists to preserve development state
COPY --from=builder /app/season.json* .

# 5. Environment & Permissions
ENV PORT=8088
ENV DATA_DIR=/app/data
ENV NODE_ENV=production

RUN chmod +x ./entrypoint.sh ./server-bin

# Pillar 4: High-Fidelity Health Monitoring
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget -qO- http://localhost:8088/api/health || exit 1

EXPOSE 8088

ENTRYPOINT ["./entrypoint.sh"]
CMD ["./server-bin"]