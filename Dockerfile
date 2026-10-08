# ==============================================================================
# Stage 1: Build Enterprise Go Orchestrator Binary
# ==============================================================================
FROM golang:1.22-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY internal/ ./internal/
COPY cmd/ ./cmd/

# Compile statically linked production binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/yuki_mongo_agent ./cmd/engine

# ==============================================================================
# Stage 2: Official MongoDB 6.0 Runtime + Cloudflared + Go Supervisor
# ==============================================================================
FROM mongo:6.0

LABEL maintainer="SUDEEPBOTS"
LABEL description="Official MongoDB 6.0 on Render with Cloudflare Tunnel & Telegram Vault"

# Install minimal system utilities for network health and certificates
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install official Cloudflare Tunnel connector
RUN curl -fsSL -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    && chmod +x /usr/local/bin/cloudflared

# Copy compiled Go engine binary from builder stage
COPY --from=builder /bin/yuki_mongo_agent /usr/local/bin/yuki_mongo_agent
RUN chmod +x /usr/local/bin/yuki_mongo_agent

# Expose Render HTTP Health Port (10000) and MongoDB Port (27017)
EXPOSE 10000 27017

# Launch pure Go orchestration engine
ENTRYPOINT ["/usr/local/bin/yuki_mongo_agent"]
