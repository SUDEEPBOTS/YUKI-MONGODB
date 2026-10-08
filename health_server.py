#!/usr/bin/env python3
"""
Yuki-Mongo-Render: HTTP Health & Keep-Alive Daemon
Satisfies Render's dynamic $PORT health checks and enables UptimeRobot 24/7 keep-alive.
"""

import os
import time
import json
import socket
import logging
import subprocess
from http.server import HTTPServer, BaseHTTPRequestHandler

logging.basicConfig(level=logging.INFO, format="%(asctime)s [HealthServer] %(message)s")
logger = logging.getLogger("HealthServer")

PORT = int(os.getenv("PORT", "10000"))
START_TIME = time.time()

def check_mongo_alive() -> bool:
    try:
        with socket.create_connection(("127.0.0.1", 27017), timeout=2):
            return True
    except Exception:
        return False

class HealthHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        mongo_status = "alive" if check_mongo_alive() else "connecting"
        uptime_sec = int(time.time() - START_TIME)

        if self.path == "/backup":
            # Trigger manual backup
            logger.info("Manual backup requested via HTTP endpoint...")
            try:
                subprocess.Popen(["python3", "/app/sync_daemon.py", "--backup"])
                msg = {"status": "success", "message": "Backup triggered in background"}
            except Exception as e:
                msg = {"status": "error", "message": str(e)}
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(msg).encode())
            return

        # Default Health / Ping
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()

        resp = {
            "status": "healthy",
            "service": "Yuki-MongoDB-Render",
            "mongodb_port": 27017,
            "mongodb_state": mongo_status,
            "uptime_seconds": uptime_sec,
            "cloud_vault": "Telegram MTProto Unlimited",
            "tunnel": "Cloudflare Zero Trust"
        }
        self.wfile.write(json.dumps(resp, indent=2).encode())

    def do_HEAD(self):
        self.send_response(200)
        self.end_headers()

    def log_message(self, format, *args):
        # Suppress spammy ping logs
        pass

def run():
    server_address = ("0.0.0.0", PORT)
    httpd = HTTPServer(server_address, HealthHandler)
    logger.info(f"🚀 Render HTTP Health server listening on 0.0.0.0:{PORT}")
    httpd.serve_forever()

if __name__ == "__main__":
    run()
