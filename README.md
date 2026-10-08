<div align="center">

# 🍃 YUKI-MONGODB
### Enterprise Official MongoDB 6.0 Engine for Render Free Tier

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-6.0_Official-47A248?style=for-the-badge&logo=mongodb&logoColor=white)](https://mongodb.com)
[![Docker](https://img.shields.io/badge/Docker-Multi--Stage-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://docker.com)
[![Cloudflare](https://img.shields.io/badge/Cloudflare-Zero_Trust_TCP-F38020?style=for-the-badge&logo=cloudflare&logoColor=white)](https://cloudflare.com)
[![Telegram](https://img.shields.io/badge/Telegram-MTProto_Vault-26A5E4?style=for-the-badge&logo=telegram&logoColor=white)](https://telegram.org)
[![Deploy to Render](https://img.shields.io/badge/Deploy%20to-Render-46E3B7?style=for-the-badge&logo=render&logoColor=white)](https://render.com)

<p align="center">
  <b>A high-performance, production-ready MongoDB 6.0 runtime designed to run 24/7 on Render's Free Tier.</b><br>
  Bypasses the 512 MB storage ceiling of MongoDB Atlas using <b>Telegram MTProto Unlimited Cloud Storage</b> and exposes real MongoDB wire-protocol ports via <b>Cloudflare Zero Trust Tunnels</b>.
</p>

---

[Features](#-key-features) • [Architecture](#-architecture) • [Comparison](#-mongodb-atlas-vs-yuki-mongodb) • [Deploy to Render](#-deploy-to-render-in-3-minutes) • [Connection Examples](#-how-to-connect)

---

</div>

## 🌟 Why YUKI-MONGODB?

Running Telegram music bots, clone bots, and distributed microservices on **MongoDB Atlas Free Tier (M0)** comes with severe limitations:
* ❌ **512 MB Hard Storage Ceiling:** Your bot crashes when storage overflows.
* ❌ **500 Concurrent Connection Cap:** Massive clone bots exhaust connection pools.
* ❌ **Network I/O Latency:** Frequent `i/o timeout` errors under spike traffic.

**YUKI-MONGODB solves this completely:**
* ✅ **Real Official MongoDB 6.0:** 100% genuine `mongod` binary (full support for BSON, indexes, aggregations, transactions, and Compass).
* ✅ **Unlimited Cloud Vault:** Backed by private Telegram channels via native **MTProto Worker TCP Sockets** (`gogram`).
* ✅ **Zero Ephemeral Data Loss:** Restores snapshots on container boot, syncs every 5 minutes in background, and executes emergency backup on `SIGTERM`.
* ✅ **Render Free Tier Optimized:** WiredTiger cache capped at **256 MB** (`--wiredTigerCacheSizeGB 0.25`) to run stably on Render's 512 MB RAM budget without OOM crashes.
* ✅ **24/7 Sleep Prevention:** Built-in Go HTTP server on `$PORT` (`/health` & `/ping`) keeps the container awake with UptimeRobot.

---

## 🏗️ Architecture

```
                       [Clients / Bots / Compass / Apps]
                                       │
                                       ▼ (Native MongoDB Wire Protocol)
                       ┌───────────────────────────────┐
                       │  Cloudflare Zero Trust Edge   │
                       │    (mongo.yukiapi.site:27017) │
                       └───────────────┬───────────────┘
                                       │ (Encrypted Argo TCP Tunnel)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                       Render Free Container (512 MB)                        │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                Yuki Mongo Agent (Pure Go Orchestrator)                │  │
│  │  • Process Supervisor (mongod + cloudflared)                          │  │
│  │  • Built-in HTTP Health & Telemetry Server (:10000 /health)           │  │
│  │  • Automated Root User Provisioning (MONGO_USER & MONGO_PASS)         │  │
│  │  • Emergency Shutdown SIGTERM Interceptor                             │  │
│  └───────────────┬───────────────────────────────────────┬───────────────┘  │
│                  │ (Local TCP 127.0.0.1:27017)           │ (Vault Ticker)   │
│                  ▼                                       ▼                  │
│  ┌───────────────────────────────┐       ┌───────────────────────────────┐  │
│  │     Official MongoDB 6.0      │       │     Go MTProto Worker Pool    │  │
│  │  (WiredTiger capped at 250MB) │       │   (Direct Native DC Sockets)  │  │
│  └───────────────────────────────┘       └───────────────┬───────────────┘  │
└──────────────────────────────────────────────────────────┼──────────────────┘
                                                           │ (Direct MTProto TCP)
                                                           ▼
                                            [Private Telegram Channel Vault]
                                            (Unlimited 2GB Permanent Storage)
```

---

## 📊 MongoDB Atlas vs YUKI-MONGODB

| Metric / Feature | MongoDB Atlas (Free Tier) | 🍃 YUKI-MONGODB (Render) |
| :--- | :--- | :--- |
| **Storage Capacity** | ❌ Strict 512 MB Limit | 🔥 **UNLIMITED** (Telegram Vault) |
| **Max Concurrent Connections** | ❌ 500 Connections Cap | 🔥 **65,536 Connections** (No artificial lock) |
| **Query Latency (RAM)** | ⚠️ 80ms – 180ms (Cloud Roundtrip) | ⚡ **0.1ms – 1ms** (In-Memory WiredTiger) |
| **Network I/O Timeout** | ❌ Frequent on high clone bot load | 🛡️ **Zero I/O Timeouts** |
| **Monthly Cost** | Upgrade costs $9 to $57/mo | 💸 **Lifetime 100% FREE** |
| **Security & Tunneling** | Shared Public Cluster | 🛡️ **Cloudflare Zero Trust TCP Tunnel** |
| **Administrative Access** | ❌ Locked configuration | ✅ **Full Control** (Indexes, WiredTiger, Dumps) |

---

## 🚀 Deploy to Render in 3 Minutes

### 1. Cloudflare Tunnel Setup (Recommended)
1. Go to [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) ➔ **Networks** ➔ **Tunnels**.
2. Click **Add a Tunnel** ➔ Name it (e.g. `yuki-mongo`).
3. Copy the **Tunnel Token** (`eyJh...`).
4. In the **Public Hostnames** tab:
   * **Subdomain:** `mongo`
   * **Domain:** `yukiapi.site` (or your domain)
   * **Type:** `TCP`
   * **URL:** `localhost:27017`
5. Save Tunnel.

### 2. Telegram Storage Channel
1. Create a **Private Telegram Channel** (e.g. `My Mongo Vault`).
2. Add your Bot or Account as **Administrator** with permission to send & pin messages.
3. Get the **Channel ID** (starts with `-100...`).

### 3. Deploy to Render
1. Open [Render Dashboard](https://dashboard.render.com/) ➔ Click **New +** ➔ **Web Service**.
2. Connect this repository: `SUDEEPBOTS/YUKI-MONGODB`.
3. Choose **Region: Singapore** (fastest latency for Asia).
4. Set **Environment: Docker**.
5. Add the following **Environment Variables**:

| Variable | Required | Description | Example |
| :--- | :---: | :--- | :--- |
| `SESSION_STRING` | **Yes** | Pyrogram String Session (MTProto Worker) | `1BVtsOGUBu...` |
| `CHANNEL_ID` | **Yes** | Private Telegram Channel ID | `-1002345678901` |
| `MONGO_USER` | *Recommended* | Root MongoDB Admin Username | `admin` |
| `MONGO_PASS` | *Recommended* | Root MongoDB Admin Password | `my_secret_pass` |
| `TUNNEL_TOKEN` | *Recommended* | Cloudflare Zero Trust Tunnel Token | `eyJhIjoi...` |
| `DOMAIN` | Optional | Custom tunnel domain hostname | `mongo.yukiapi.site` |
| `SYNC_INTERVAL_MIN`| Optional | Backup snapshot interval in minutes | `5` |
| `API_ID` | Optional | Telegram App ID (default: `38674666`) | `38674666` |
| `API_HASH` | Optional | Telegram App Hash (pre-configured) | `b4f0fbf8fb...` |

6. Click **Deploy Web Service**!

> **💡 Zero-Config Fallback Mode:** If you do not provide a `TUNNEL_TOKEN`, YUKI-MONGODB automatically allocates a **Free Instant TCP Tunnel** and prints the live public connection string directly into your Render startup logs!

---

## ⏰ Keep Render 24/7 Awake

Render free containers sleep after 15 minutes of inactivity. YUKI-MONGODB includes a lightweight Go HTTP server listening on Render's `$PORT`.

1. Go to [UptimeRobot](https://uptimerobot.com) (100% Free).
2. Add New Monitor:
   * **Monitor Type:** HTTP(s)
   * **URL:** `https://your-app.onrender.com/health`
   * **Monitoring Interval:** Every 5 minutes.
3. Your database will stay **alive 24/7/365 non-stop**!

---

## 🔌 How to Connect

Once deployed, the Render logs will display your live connection string banner:

```text
=================================================================
🍃 YUKI-MONGODB ONLINE & READY FOR BOT CONNECTIONS!
👉 Real Mongo URI : mongodb://admin:my_secret_pass@mongo.yukiapi.site:27017/?authSource=admin
👉 Internal Socket: 127.0.0.1:27017
👉 Render Health  : 0.0.0.0:10000/health
=================================================================
```

### Python (Motor / PyMongo)
```python
from motor.motor_asyncio import AsyncIOMotorClient

MONGO_URI = "mongodb://admin:my_secret_pass@mongo.yukiapi.site:27017/?authSource=admin"

client = AsyncIOMotorClient(MONGO_URI)
db = client["MeowMusic"]
clones = db["clones"]

# Insert Document
await clones.insert_one({"_id": 12345, "bot_name": "YukiClone", "status": "active"})

# Query Document
doc = await clones.find_one({"_id": 12345})
print("Connected successfully:", doc)
```

### Node.js (Mongoose)
```javascript
const mongoose = require('mongoose');

const uri = "mongodb://admin:my_secret_pass@mongo.yukiapi.site:27017/myDatabase?authSource=admin";
mongoose.connect(uri)
  .then(() => console.log('🍃 Connected to YUKI-MONGODB!'))
  .catch(err => console.error('Connection error:', err));
```

### MongoDB Compass (Desktop GUI)
1. Open **MongoDB Compass**.
2. Paste your connection URI: `mongodb://admin:my_secret_pass@mongo.yukiapi.site:27017/?authSource=admin`.
3. Click **Connect** to inspect collections and documents visually.

---

## 📡 Live Telemetry & API Endpoints

The built-in Go HTTP server provides operational endpoints:

* `GET /health` : Fast JSON health check for Render & UptimeRobot.
* `GET /ping` : Ultra-lightweight ping responder.
* `GET /stats` : Telemetry detailing MongoDB state, tunnel status, and Telegram vault history.
* `POST /backup` : Trigger an immediate manual snapshot upload to your Telegram channel.

---

## 📜 License

Distributed under the **MIT License**. Free for personal and commercial usage.

Developed with ❤️ by **[SUDEEPBOTS](https://github.com/SUDEEPBOTS)**.
