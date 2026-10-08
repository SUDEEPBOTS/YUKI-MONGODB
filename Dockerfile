# =================================================================================================
# Project       : YUKI-MONGODB
# Official Repo : https://github.com/SUDEEPBOTS/YUKI-MONGODB
# Description   : Multi-Stage Production Container for MongoDB 6.0 & Yuki Agent
# Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
#
# Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
# Licensed under the MIT License (https://opensource.org/licenses/MIT)
# =================================================================================================

FROM golang:1.27-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY internal/ ./internal/
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/yuki_mongo_agent ./cmd/engine

FROM mongo:6.0
LABEL maintainer="SUDEEPBOTS"
LABEL description="Official MongoDB 6.0 on Render with Cloudflare Tunnel & Unlimited Cloud Vault"

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    openssh-client \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    && chmod +x /usr/local/bin/cloudflared

COPY --from=builder /bin/yuki_mongo_agent /usr/local/bin/yuki_mongo_agent
RUN chmod +x /usr/local/bin/yuki_mongo_agent

EXPOSE 10000

ENTRYPOINT ["/usr/local/bin/yuki_mongo_agent"]
