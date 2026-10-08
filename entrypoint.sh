#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "🚀 Initializing Yuki-MongoDB on Render"
echo "=========================================================="

# 1. Prepare MongoDB Data Directory
mkdir -p /data/db
chmod 777 /data/db

# 2. Launch MongoDB with WiredTiger Cache capped at 250MB (Crucial for Render 512MB RAM tier)
echo "[1/4] Starting MongoDB engine on 127.0.0.1:27017 (WiredTiger capped at 256MB)..."
mongod --bind_ip_all --port 27017 --wiredTigerCacheSizeGB 0.25 &
MONGO_PID=$!

# Wait for MongoDB to accept connections
echo "[2/4] Waiting for MongoDB socket to be ready..."
for i in {1..30}; do
    if nc -z 127.0.0.1 27017 2>/dev/null || (echo > /dev/tcp/127.0.0.1/27017) 2>/dev/null; then
        echo "✅ MongoDB is online and listening on 27017!"
        break
    fi
    sleep 0.5
done

# 3. Restore Latest Snapshot from Telegram Vault (if BOT_TOKEN & CHANNEL_ID set)
if [ -n "$BOT_TOKEN" ] && [ -n "$CHANNEL_ID" ]; then
    echo "[3/4] 📥 Checking Telegram Vault for latest database snapshot..."
    python3 /app/sync_daemon.py --restore || echo "[WARN] Restore encountered non-fatal notice, continuing..."
    # Start periodic background sync daemon (every 5-10 minutes)
    python3 /app/sync_daemon.py --loop &
else
    echo "[3/4] ⚠️ BOT_TOKEN or CHANNEL_ID not provided. Running without Telegram auto-restore."
fi

# 4. Launch Cloudflare Tunnel
if [ -n "$TUNNEL_TOKEN" ]; then
    echo "[4/4] ⚡ Starting Cloudflare Tunnel daemon..."
    cloudflared tunnel --no-autoupdate run --token "$TUNNEL_TOKEN" &
else
    echo "[4/4] ⚠️ TUNNEL_TOKEN not provided. Cloudflare Tunnel is inactive."
fi

echo "=========================================================="
echo "🎯 Yuki-MongoDB Engine is fully operational!"
echo "   • Internal Mongo Port: 27017"
echo "   • Render Health Port : ${PORT:-10000}"
echo "=========================================================="

# 5. Start HTTP Health & Keep-Alive Server as foreground process
exec python3 /app/health_server.py
