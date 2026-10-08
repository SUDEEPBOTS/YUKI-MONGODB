FROM mongo:6.0

# Install dependencies for Cloudflare Tunnel, Telegram Sync, and Python
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    wget \
    ca-certificates \
    python3 \
    netcat-openbsd \
    tar \
    gzip \
    && rm -rf /var/lib/apt/lists/*

# Download official Cloudflare Tunnel (cloudflared) binary
RUN curl -fsSL -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    && chmod +x /usr/local/bin/cloudflared

WORKDIR /app

# Copy scripts
COPY entrypoint.sh /app/entrypoint.sh
COPY sync_daemon.py /app/sync_daemon.py
COPY health_server.py /app/health_server.py

RUN chmod +x /app/entrypoint.sh /app/sync_daemon.py /app/health_server.py

# Expose Render HTTP health port and Mongo port
EXPOSE 10000 27017

ENTRYPOINT ["/app/entrypoint.sh"]
