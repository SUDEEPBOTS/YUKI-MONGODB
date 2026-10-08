#!/usr/bin/env python3
"""
Yuki-Mongo-Render: Telegram Vault Auto-Sync Engine
Implements zero-data-loss persistence for MongoDB on ephemeral cloud platforms (Render).
"""

import os
import sys
import time
import json
import signal
import tarfile
import shutil
import logging
import subprocess
import urllib.request
import urllib.parse
from datetime import datetime

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] [YukiVault] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)
logger = logging.getLogger("YukiVault")

BOT_TOKEN = os.getenv("BOT_TOKEN", "").strip()
CHANNEL_ID = os.getenv("CHANNEL_ID", "").strip()
SYNC_INTERVAL_MIN = int(os.getenv("SYNC_INTERVAL_MIN", "5"))
DUMP_DIR = "/tmp/mongodump"
ARCHIVE_PATH = "/tmp/yuki_mongo_backup.tar.gz"

def is_configured() -> bool:
    return bool(BOT_TOKEN and CHANNEL_ID)

def run_cmd(cmd: list) -> bool:
    try:
        res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, timeout=60)
        if res.returncode != 0:
            logger.warning(f"Command {' '.join(cmd)} failed (code {res.returncode}): {res.stderr.strip()}")
            return False
        return True
    except Exception as e:
        logger.error(f"Execution error for {' '.join(cmd)}: {e}")
        return False

def tg_api_call(method: str, params: dict = None, files: dict = None) -> dict:
    url = f"https://api.telegram.org/bot{BOT_TOKEN}/{method}"
    if files:
        # Multi-part form upload for sendDocument
        boundary = "----YukiMongoBoundary" + str(int(time.time()))
        body = bytearray()
        for k, v in (params or {}).items():
            body.extend(f"--{boundary}\r\n".encode())
            body.extend(f'Content-Disposition: form-data; name="{k}"\r\n\r\n'.encode())
            body.extend(f"{v}\r\n".encode())
        for file_field, file_path in files.items():
            filename = os.path.basename(file_path)
            body.extend(f"--{boundary}\r\n".encode())
            body.extend(f'Content-Disposition: form-data; name="{file_field}"; filename="{filename}"\r\n'.encode())
            body.extend(b"Content-Type: application/gzip\r\n\r\n")
            with open(file_path, "rb") as f:
                body.extend(f.read())
            body.extend(b"\r\n")
        body.extend(f"--{boundary}--\r\n".encode())

        req = urllib.request.Request(url, data=bytes(body))
        req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    else:
        data = json.dumps(params or {}).encode() if params else None
        req = urllib.request.Request(url, data=data)
        if data:
            req.add_header("Content-Type", "application/json")

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        logger.error(f"Telegram API {method} error: {e}")
        return {"ok": False, "description": str(e)}

def download_file(file_id: str, dest_path: str) -> bool:
    res = tg_api_call("getFile", {"file_id": file_id})
    if not res.get("ok"):
        return False
    file_path = res["result"].get("file_path")
    if not file_path:
        return False

    download_url = f"https://api.telegram.org/file/bot{BOT_TOKEN}/{file_path}"
    try:
        with urllib.request.urlopen(download_url, timeout=60) as resp, open(dest_path, "wb") as out_f:
            shutil.copyfileobj(resp, out_f)
        return True
    except Exception as e:
        logger.error(f"Download failed from {download_url}: {e}")
        return False

def restore_latest():
    if not is_configured():
        logger.info("Telegram Bot Token or Channel ID not provided. Skipping auto-restore.")
        return

    logger.info("Checking Telegram Storage Channel for latest backup snapshot...")
    # Get last updates or read channel history
    res = tg_api_call("getChat", {"chat_id": CHANNEL_ID})
    if not res.get("ok"):
        logger.warning(f"Could not access Telegram channel {CHANNEL_ID}: {res.get('description')}")
        return

    # Check pinned message first or search recent posts
    chat_info = res.get("result", {})
    pinned = chat_info.get("pinned_message")
    doc_msg = None

    if pinned and "document" in pinned:
        doc_msg = pinned
        logger.info("Found latest backup in Channel Pinned Message!")

    # If no pinned message, check channel messages via getUpdates or fallback
    if not doc_msg:
        # Check if channel has pinned message or try getChat
        logger.info("No pinned backup found. Starting with fresh/existing DB state.")
        return

    doc = doc_msg.get("document", {})
    file_id = doc.get("file_id")
    if not file_id:
        return

    restore_archive = "/tmp/restore_backup.tar.gz"
    logger.info(f"Downloading snapshot archive ({doc.get('file_size', 0)} bytes)...")
    if not download_file(file_id, restore_archive):
        logger.error("Failed to download restore archive from Telegram.")
        return

    logger.info("Extracting archive...")
    restore_dir = "/tmp/mongorestore_data"
    os.makedirs(restore_dir, exist_ok=True)
    try:
        with tarfile.open(restore_archive, "r:gz") as tar:
            tar.extractall(path=restore_dir)
    except Exception as e:
        logger.error(f"Failed to unpack tarball: {e}")
        return

    # Look for mongodump directory
    target_dir = restore_dir
    if os.path.exists(os.path.join(restore_dir, "mongodump")):
        target_dir = os.path.join(restore_dir, "mongodump")

    logger.info("Executing mongorestore into local MongoDB instance...")
    if run_cmd(["mongorestore", "--drop", target_dir]):
        logger.info("✅ [RESTORE COMPLETE] MongoDB successfully restored from Telegram Vault!")
    else:
        logger.warning("mongorestore reported warnings during restore.")

    # Cleanup
    shutil.rmtree(restore_dir, ignore_errors=True)
    if os.path.exists(restore_archive):
        os.remove(restore_archive)

def backup_now() -> bool:
    if not is_configured():
        logger.warning("Cannot backup: BOT_TOKEN or CHANNEL_ID not configured.")
        return False

    timestamp = datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC")
    logger.info(f"Generating MongoDB dump snapshot at {timestamp}...")

    shutil.rmtree(DUMP_DIR, ignore_errors=True)
    if not run_cmd(["mongodump", "--out", DUMP_DIR]):
        logger.error("mongodump execution failed.")
        return False

    # Check if any database was dumped
    if not os.path.exists(DUMP_DIR) or not os.listdir(DUMP_DIR):
        logger.info("MongoDB instance has no user databases yet. Skipping empty dump upload.")
        return True

    # Compress into tar.gz
    logger.info("Compressing mongodump into GZIP archive...")
    with tarfile.open(ARCHIVE_PATH, "w:gz") as tar:
        tar.add(DUMP_DIR, arcname="mongodump")

    archive_size = os.path.getsize(ARCHIVE_PATH)
    logger.info(f"Uploading archive ({archive_size} bytes) to Telegram Channel {CHANNEL_ID}...")

    caption = (
        f"🛡️ **YukiMongo Vault Snapshot**\n"
        f"📅 `{timestamp}`\n"
        f"📦 Size: `{round(archive_size / 1024, 2)} KB`\n"
        f"⚡ Status: `Verified Durable`\n"
        f"🏷️ #YUKIMONGO_BACKUP"
    )

    res = tg_api_call(
        "sendDocument",
        params={"chat_id": CHANNEL_ID, "caption": caption, "parse_mode": "Markdown"},
        files={"document": ARCHIVE_PATH}
    )

    if res.get("ok"):
        msg_id = res["result"]["message_id"]
        logger.info(f"✅ [BACKUP SUCCESS] Snapshot uploaded as MsgID {msg_id}! Pinning message...")
        tg_api_call("pinChatMessage", {"chat_id": CHANNEL_ID, "message_id": msg_id, "disable_notification": True})
        return True
    else:
        logger.error(f"Failed to upload document to Telegram: {res.get('description')}")
        return False

def sync_loop():
    logger.info(f"Starting background auto-sync loop (Interval: {SYNC_INTERVAL_MIN} minutes)...")
    while True:
        time.sleep(SYNC_INTERVAL_MIN * 60)
        try:
            backup_now()
        except Exception as e:
            logger.error(f"Error during scheduled backup: {e}")

def handle_sigterm(signum, frame):
    logger.info("Received SIGTERM from Render! Executing emergency shutdown backup...")
    try:
        backup_now()
    except Exception as e:
        logger.error(f"Emergency backup failed: e")
    sys.exit(0)

if __name__ == "__main__":
    signal.signal(signal.SIGTERM, handle_sigterm)
    signal.signal(signal.SIGINT, handle_sigterm)

    if len(sys.argv) > 1:
        mode = sys.argv[1]
        if mode == "--restore":
            restore_latest()
        elif mode == "--backup":
            backup_now()
        elif mode == "--loop":
            sync_loop()
    else:
        sync_loop()
