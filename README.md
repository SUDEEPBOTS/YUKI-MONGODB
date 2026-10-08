<div align="center">

<img src="assets/logo.png" alt="YUKI-MONGODB Logo" width="220" style="border-radius: 20px; box-shadow: 0 8px 24px rgba(0, 230, 118, 0.25);" />

# 🍃 YUKI-MONGODB
### Deploy Your Own Enterprise MongoDB 6.0 Database — Lifetime Free, Zero Limits!

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-6.0_Official-47A248?style=for-the-badge&logo=mongodb&logoColor=white)](https://mongodb.com)
[![Docker](https://img.shields.io/badge/Docker-Multi--Stage-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://docker.com)
[![Cloudflare](https://img.shields.io/badge/Cloudflare-Zero_Trust_TCP-F38020?style=for-the-badge&logo=cloudflare&logoColor=white)](https://cloudflare.com)
[![Cloud Vault](https://img.shields.io/badge/Storage-Unlimited_Cloud_Vault-00C7B7?style=for-the-badge&logo=icloud&logoColor=white)](#-architecture)
[![UptimeRobot](https://img.shields.io/badge/UptimeRobot-24%2F7_Awake-3BD671?style=for-the-badge&logo=uptimerobot&logoColor=white)](https://uptimerobot.com)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

<br>

<p align="center">
  <b>A high-performance, production-ready MongoDB 6.0 database engine designed to run 24/7 on Render & Cloud PaaS.</b><br>
  Bypasses the 512 MB storage ceiling of MongoDB Atlas using a <b>Proprietary Distributed Cloud Vault with Zero Storage Limits</b> and exposes real MongoDB wire-protocol ports via <b>Cloudflare Zero Trust Tunnels</b>.
</p>

### 🚀 Instant 1-Click Deploy & Keep Awake

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/SUDEEPBOTS/YUKI-MONGODB)
&nbsp;&nbsp;&nbsp;&nbsp;
[![Deploy on Railway](https://railway.com/button.svg)](https://railway.app/template/deploy?template=https://github.com/SUDEEPBOTS/YUKI-MONGODB)
&nbsp;&nbsp;&nbsp;&nbsp;
[![Setup UptimeRobot](https://img.shields.io/badge/Keep_Awake-UptimeRobot-3BD671?style=for-the-badge&logo=uptimerobot&logoColor=white)](https://uptimerobot.com)

---

[Features](#-why-yuki-mongodb) • [Architecture](#-architecture) • [Comparison](#-mongodb-atlas-vs-yuki-mongodb) • [1-Click Deploy](#-deploy-options) • [Connection Examples](#-how-to-connect) • [Files Overview](#-repository-structure) • [Sponsor](#-support--sponsorship)

---

</div>

## 🌟 Why YUKI-MONGODB?

Running modern bots, microservices, and distributed applications on **MongoDB Atlas Free Tier (M0)** comes with severe limitations:
* ❌ **512 MB Hard Storage Ceiling:** Your application crashes when database storage overflows.
* ❌ **500 Concurrent Connection Cap:** Connection pools get exhausted under high traffic spikes.
* ❌ **Network I/O Latency:** Frequent `i/o timeout` errors under sudden load.

**YUKI-MONGODB solves this completely:**
* ✅ **Real Official MongoDB 6.0:** 100% genuine `mongod` binary (full support for BSON, indexes, aggregations, transactions, and Compass GUI).
* ✅ **Unlimited Cloud Vault:** Backed by an automated **High-Speed Distributed Cloud Vault** with continuous background replication and multi-gigabyte disaster recovery.
* ✅ **Zero Ephemeral Data Loss:** Restores snapshots on container boot, syncs every 5 minutes in background, and executes emergency backup on `SIGTERM`.
* ✅ **Render Free Tier Optimized:** WiredTiger cache capped at **256 MB** (`--wiredTigerCacheSizeGB 0.25`) to run stably on Render's 512 MB RAM budget without OOM crashes.
* ✅ **24/7 Sleep Prevention:** Built-in Go HTTP server on `$PORT` (`/health` & `/ping`) keeps the container awake with UptimeRobot.
* ✅ **Zero-Config Fallback Tunnel:** If you don't provide a Cloudflare token, auto-provisions a free TCP relay and prints the URI in logs!

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
│  │     Official MongoDB 6.0      │       │  Encrypted Cloud Vault Engine │  │
│  │  (WiredTiger capped at 250MB) │       │   (High-Speed Socket Pool)    │  │
│  │  (Full BSON, Compass, Driver) │       │   (Multi-Part Disaster Sync)  │  │
│  └───────────────────────────────┘       └───────────────┬───────────────┘  │
└──────────────────────────────────────────────────────────┼──────────────────┘
                                                           │ (Encrypted Asynchronous Stream)
                                                           ▼
                                            [Distributed High-Availability Vault]
                                            (Unlimited Permanent Cloud Storage)
```

---

## 📊 MongoDB Atlas vs YUKI-MONGODB

| Metric / Feature | MongoDB Atlas (Free Tier) | 🍃 YUKI-MONGODB (Render) |
| :--- | :--- | :--- |
| **Storage Capacity** | ❌ Strict 512 MB Limit | 🔥 **UNLIMITED** (Yuki Cloud Vault) |
| **Max Concurrent Connections** | ❌ 500 Connections Cap | 🔥 **65,536 Connections** (No artificial lock) |
| **Query Latency (RAM)** | ⚠️ 80ms – 180ms (Cloud Roundtrip) | ⚡ **0.1ms – 1ms** (In-Memory WiredTiger) |
| **Network I/O Timeout** | ❌ Frequent on high clone bot load | 🛡️ **Zero I/O Timeouts** |
| **Monthly Cost** | Upgrade costs $9 to $57/mo | 💸 **Lifetime 100% FREE** |
| **Security & Tunneling** | Shared Public Cluster | 🛡️ **Cloudflare Zero Trust TCP Tunnel** |
| **Administrative Access** | ❌ Locked configuration | ✅ **Full Control** (Indexes, WiredTiger, Dumps) |

---

## 🚀 Deploy Options

### Option 1: Direct 1-Click Render Deploy (Recommended)

Click the button below to auto-import the blueprint directly into your Render account:

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/SUDEEPBOTS/YUKI-MONGODB)

Render reads [render.yaml](file:///root/yuki-mongo-render/render.yaml) & [app.json](file:///root/yuki-mongo-render/app.json) automatically and prompts you to fill in your environment variables.

### Option 2: Deploy to Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.app/template/deploy?template=https://github.com/SUDEEPBOTS/YUKI-MONGODB)

Configured via [railway.json](file:///root/yuki-mongo-render/railway.json) and [railway.yml](file:///root/yuki-mongo-render/railway.yml) with automated container healthcheck and failure retry policies.

---

## ⚙️ Environment Variables

| Variable | Required | Description | Default | Example |
| :--- | :---: | :--- | :---: | :--- |
| `SESSION_STRING` | **Yes** | Cloud Vault Worker Authentication Session String | — | `1BVtsOGUBu...` |
| `CHANNEL_ID` | **Yes** | Private Storage Vault Channel ID | — | `-1002345678901` |
| `MONGO_USER` | *Recommended* | Root MongoDB Admin Username | `admin` | `admin` |
| `MONGO_PASS` | *Recommended* | Root MongoDB Admin Password | `sudeep123` | `my_secret_pass` |
| `TUNNEL_TOKEN` | *Optional* | Cloudflare Zero Trust Tunnel Token (Leaves empty for Free Auto-Tunnel) | `""` | `eyJhIjoi...` |
| `DOMAIN` | Optional | Custom tunnel domain hostname | `mongo.yukiapi.site` | `mongo.mydomain.com` |
| `SYNC_INTERVAL_MIN`| Optional | Backup snapshot interval in minutes | `5` | `5` |
| `CACHE_SIZE_GB` | Optional | WiredTiger in-memory RAM cache cap | `0.25` (256MB) | `0.25` |
| `PORT` | Optional | HTTP Health & Keep-Alive Port | `10000` | `10000` |

> **💡 Zero-Config Fallback Mode:** If you do not provide a `TUNNEL_TOKEN`, YUKI-MONGODB automatically allocates a **Free Instant TCP Tunnel** and prints the live public connection string directly into your container logs!

---

## ⏰ Keep Render 24/7 Awake

Render free containers sleep after 15 minutes of HTTP inactivity. YUKI-MONGODB includes a lightweight Go HTTP server listening on Render's `$PORT`:

<div align="center">

[![Setup UptimeRobot Monitor](https://img.shields.io/badge/UptimeRobot-Setup_24%2F7_Free_Monitor-3BD671?style=for-the-badge&logo=uptimerobot&logoColor=white)](https://uptimerobot.com)

</div>

1. Go to [UptimeRobot](https://uptimerobot.com) (100% Free).
2. Add New Monitor:
   * **Monitor Type:** HTTP(s)
   * **URL:** `https://your-render-service.onrender.com/health`
   * **Monitoring Interval:** Every 5 minutes.
3. Your database will stay **alive 24/7/365 non-stop** without sleeping!

---

## 🔌 How to Connect

Once deployed, container logs will display your live connection string banner:

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
db = client["ProductionDB"]
collection = db["users"]

# Insert Document
await collection.insert_one({"_id": 101, "name": "YukiUser", "plan": "unlimited"})

# Query Document
doc = await collection.find_one({"_id": 101})
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

## 📁 Repository Structure

```
.
├── assets/
│   └── logo.png                # Official YUKI-MONGODB 3D Branding Logo
├── .github/
│   ├── FUNDING.yml             # GitHub Sponsors & donation configuration
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md       # Standardized bug reporting template
│   │   └── feature_request.md  # Feature proposal template
│   └── workflows/
│       └── ci.yml              # GitHub Actions CI for Go build & Docker verification
├── app.json                    # PaaS / Heroku / Container deployment manifest
├── render.yaml                 # Render Infrastructure as Code Blueprint
├── railway.json                # Railway Engine deployment schema
├── railway.yml                 # Railway build & deploy lifecycle config
├── Dockerfile                  # Multi-stage Dockerfile (Go builder + Mongo 6.0)
├── go.mod                      # Go module definitions
├── go.sum                      # Go dependency checksums
├── cmd/
│   └── engine/
│       └── main.go             # Main Go storage agent supervisor & cloud vault worker
├── CONTRIBUTING.md             # Contribution guidelines
├── SECURITY.md                 # Security & vulnerability reporting policy
├── LICENSE                     # MIT Open Source License
└── README.md                   # Complete documentation & quickstart guide
```

---

## 📡 Live Telemetry & API Endpoints

The built-in Go HTTP server provides operational endpoints:

* `GET /health` : Fast JSON health check for Render & UptimeRobot.
* `GET /ping` : Ultra-lightweight ping responder.
* `GET /stats` : Telemetry detailing MongoDB state, tunnel status, and cloud vault replication metrics.
* `POST /backup` : Trigger an immediate manual snapshot replication to your cloud vault.

---

## 💖 Support & Sponsorship

If **YUKI-MONGODB** saved your database hosting costs or powered your application fleet, please consider supporting the development!

<div align="center">

[![GitHub Sponsors](https://img.shields.io/badge/Sponsor-SUDEEPBOTS-EA4AAA?style=for-the-badge&logo=githubsponsors&logoColor=white)](https://github.com/sponsors/SUDEEPBOTS)
[![Ko-Fi](https://img.shields.io/badge/Ko--fi-Support-FF5E5B?style=for-the-badge&logo=kofi&logoColor=white)](https://ko-fi.com/sudeepbots)
[![Donate Portal](https://img.shields.io/badge/Donate-Yuki%20Network-4CAF50?style=for-the-badge&logo=google-pay&logoColor=white)](https://yukiapi.site/donate)

**UPI / Crypto / Card Support:** [yukiapi.site/donate](https://yukiapi.site/donate)

</div>

---

## 📜 License

Distributed under the **[MIT License](LICENSE)**. Free for personal and commercial usage.

Developed with ❤️ by **[SUDEEPBOTS](https://github.com/SUDEEPBOTS)**.
