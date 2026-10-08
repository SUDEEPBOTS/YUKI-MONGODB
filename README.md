# 🍃 Yuki-Mongo-Render (Enterprise Go Engine)

**Official MongoDB 6.0 on Render Free Tier with Cloudflare Tunnel & Telegram Unlimited Persistent Vault.**

Zero Python. Built entirely in pure **Golang** with concurrent process supervision, native Telegram Bot API integration, GZIP archive management, and Cloudflare Zero Trust tunnel orchestration.

---

## 🏗️ Architecture

```
                       [Clients / Bots / Compass]
                                   │
                                   ▼ (Protected TCP Wire Protocol via Cloudflare)
                   ┌─────────────────────────────────┐
                   │   Cloudflare Zero Trust Edge    │
                   │     (mongo.yukiapi.site)        │
                   └────────────────┬────────────────┘
                                    │ (Encrypted Argo Tunnel)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        Render Free Container                           │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                Yuki Mongo Agent (Go Orchestrator)                │  │
│  │  • Process Supervisor (mongod + cloudflared)                     │  │
│  │  • Render Health Server (:10000 /health /ping /stats)            │  │
│  │  • Graceful Shutdown SIGTERM Interceptor                         │  │
│  └───────────────┬───────────────────────────────┬──────────────────┘  │
│                  │ (Local Socket)                │ (Vault Ticker)      │
│                  ▼                               ▼                     │
│  ┌───────────────────────────────┐  ┌───────────────────────────────┐  │
│  │     Official MongoDB 6.0      │  │      Telegram Auto-Sync       │  │
│  │  (WiredTiger capped at 250MB) │  │  (Native GZIP Compression)    │  │
│  └───────────────────────────────┘  └───────────────┬───────────────┘  │
└─────────────────────────────────────────────────────┼──────────────────┘
                                                      │ (HTTPS Multipart)
                                                      ▼
                                       [Private Telegram Channel]
                                      (Unlimited Cloud Vault 24/7)
```

---

## ⚡ Key Highlights
* **Zero Python Overhead:** Compiled directly to a single Go static binary (`yuki_mongo_agent`) consuming <10 MB RAM.
* **Real Official MongoDB 6.0:** 100% wire-protocol compatibility with `pymongo`, `motor`, `mongoose`, and MongoDB Compass.
* **Render RAM Protection:** WiredTiger cache capped at **256 MB** (`--wiredTigerCacheSizeGB 0.25`) to prevent Render 512 MB Out-Of-Memory (OOM) kills.
* **Zero Data Loss (Yukisbox Technique):**
  * **On Boot:** Downloads and restores the latest snapshot from your Telegram Channel.
  * **Scheduled Ticker:** Takes regular `mongodump` snapshots, compresses to GZIP, uploads to Telegram, and auto-pins.
  * **Emergency Shutdown:** Hooks OS `SIGTERM` / `SIGINT` from Render to push a final emergency backup before container shutdown.
* **Cloudflare Zero Trust Tunnel:** Directly pipes MongoDB Port 27017 through Cloudflare's secure network without exposing raw VPS ports.
* **Render Keep-Alive Ready:** Built-in HTTP server listening on Render's `$PORT` (`/health` & `/ping`) for UptimeRobot keep-alives.

---

## 🛠️ Step 1: Cloudflare Tunnel Setup (2 Minutes)

1. Open [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) ➔ **Networks** ➔ **Tunnels**.
2. Click **Add a Tunnel** ➔ Name it (e.g. `render-mongo`).
3. Under **Install connector**, copy your **Tunnel Token** (`eyJh...`).
4. Click **Next** ➔ **Public Hostnames** tab:
   * **Subdomain:** `mongo`
   * **Domain:** `yukiapi.site` (or your domain)
   * **Type:** `TCP`
   * **URL:** `localhost:27017`
5. Save Tunnel.

---

## 📱 Step 2: Telegram Storage Channel Setup (1 Minute)

1. Create a **Private Telegram Channel** (e.g. `My Mongo Vault`).
2. Add your Telegram Bot as **Administrator** with permission to send messages and pin messages.
3. Obtain the Channel ID (starts with `-100...`).

---

## 🚀 Step 3: Deploy to Render (Singapore)

1. Push this repository to your GitHub:
   ```bash
   git add .
   git commit -m "feat: enterprise go mongo orchestrator"
   git push -u origin main
   ```
2. In [Render Dashboard](https://dashboard.render.com/):
   * Click **New +** ➔ **Web Service**.
   * Connect your repository.
   * Environment: **Docker**.
   * Region: **Singapore** (recommended for lowest latency in Asia).
   * Plan: **Free**.
3. Add Environment Variables:
   | Key | Value | Description |
   | :--- | :--- | :--- |
   | `BOT_TOKEN` | `123456:ABC...` | Telegram Bot Token from @BotFather |
   | `CHANNEL_ID` | `-100xxxxxxxxx` | Private Telegram Channel ID |
   | `TUNNEL_TOKEN` | `eyJh...` | Cloudflare Tunnel Token from Step 1 |
   | `SYNC_INTERVAL_MIN` | `5` | Backup frequency in minutes (Default: 5) |
4. Set Health Check Path: `/health`.
5. Click **Deploy Web Service**!

---

## ⏰ Step 4: Keep Render 24/7 Awake

1. Go to [UptimeRobot](https://uptimerobot.com) (Free).
2. Add New Monitor:
   * **Type:** HTTP(s)
   * **URL:** `https://your-app.onrender.com/health`
   * **Interval:** Every 5 minutes.
3. Your Render container will never go to sleep!

---

## 🔌 Step 5: Connecting Your Bots & Tools

Because Cloudflare protects TCP traffic, external machines route through Cloudflare's lightweight local connector:

### On your Bot Server (VPS / Cloud / Machine):
Run this background command once:
```bash
cloudflared access tcp --hostname mongo.yukiapi.site --url 127.0.0.1:27017 &
```

### In your Bot Code:
Connect directly to localhost standard MongoDB:
```python
# Python (Motor / PyMongo)
MONGO_URI = "mongodb://127.0.0.1:27017"
```

```javascript
// Node.js (Mongoose)
const mongoose = require('mongoose');
mongoose.connect('mongodb://127.0.0.1:27017/myDatabase');
```

---

## 📊 Endpoints & Telemetry
* `GET /health` : JSON health status for Render and UptimeRobot
* `GET /ping` : Quick lightweight ping
* `GET /stats` : Telemetry detailing MongoDB state, Cloudflared status, and Vault sync history
* `POST /backup` : Trigger an immediate manual database backup to Telegram
