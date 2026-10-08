# 🍃 Yuki-Mongo-Render

**Official MongoDB 6.0 Engine on Render Free Tier with Cloudflare Tunnel & Telegram Unlimited Storage Vault.**

Eliminates the 512 MB ceiling of MongoDB Atlas and overcomes Render's ephemeral disk wipes using Telegram MTProto unlimited storage backups.

---

## ⚡ Key Highlights
* **Real Official MongoDB:** Pure `mongod` 6.0 binary (100% full PyMongo / Motor / Compass compatibility).
* **RAM Capped for Render:** Configured with `--wiredTigerCacheSizeGB 0.25` to stay comfortably within Render's 512 MB free tier memory cap.
* **Zero Data Loss (Yukisbox Technique):**
  * **On Boot:** Automatically restores the latest database snapshot from your private Telegram Storage Channel.
  * **Background Loop:** Takes periodic GZIP compressed `mongodump` snapshots and uploads them to Telegram.
  * **On Shutdown:** Catches Render's `SIGTERM` and executes an emergency backup before shutdown.
* **Cloudflare Tunnel (TCP Port 27017):** Exposes MongoDB safely via Cloudflare Zero Trust with DDoS protection and without open ports.
* **24/7 Keep-Alive:** Built-in HTTP server listening on Render's `$PORT` (`/health` endpoint) for UptimeRobot pings.

---

## 🛠️ Step 1: Cloudflare Tunnel Setup (2 Minutes)

1. Open [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) -> **Networks** -> **Tunnels**.
2. Click **Add a Tunnel** -> Name it (e.g. `render-mongo`).
3. Under **Install connector**, copy your **Tunnel Token** (starts with `eyJh...`).
4. Click **Next** -> **Public Hostnames** tab:
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

1. Push this folder to your GitHub:
   ```bash
   git init
   git add .
   git commit -m "feat: initial yuki-mongo-render setup"
   git branch -M main
   git remote add origin https://github.com/YOUR_USERNAME/yuki-mongo-render.git
   git push -u origin main
   ```
2. In [Render Dashboard](https://dashboard.render.com/):
   * Click **New +** -> **Web Service**.
   * Connect your GitHub repository.
   * Environment: **Docker**.
   * Region: **Singapore** (recommended for low latency).
   * Plan: **Free**.
3. Add Environment Variables:
   | Key | Value | Description |
   | :--- | :--- | :--- |
   | `BOT_TOKEN` | `123456:ABC...` | Telegram Bot Token from @BotFather |
   | `CHANNEL_ID` | `-100xxxxxxxxx` | Private Telegram Channel ID |
   | `TUNNEL_TOKEN` | `eyJh...` | Cloudflare Tunnel Token from Step 1 |
   | `SYNC_INTERVAL_MIN` | `5` | Backup frequency in minutes (Default: 5) |
4. Set Health Check Path to: `/health`.
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

## 🔌 Step 5: Connecting Bots to your MongoDB

Because Cloudflare protects TCP traffic, external clients (like your VPS or laptop) route through Cloudflare's lightweight local connector:

### On your VPS / Bot Server:
Run this background command once:
```bash
cloudflared access tcp --hostname mongo.yukiapi.site --url 127.0.0.1:27017 &
```

### In your Bot Code (`config.py`):
Connect to MongoDB locally like normal:
```python
MONGO_URI = "mongodb://127.0.0.1:27017"
```

Any Python bot (`motor`, `pymongo`), Node.js app (`mongoose`), or MongoDB Compass will now connect with full native MongoDB speed!
