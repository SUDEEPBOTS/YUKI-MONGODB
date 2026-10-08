package vault

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
)

type SyncerStats struct {
	LastSyncTime    time.Time `json:"last_sync_time"`
	LastArchiveSize int64     `json:"last_archive_size_bytes"`
	SyncCount       int64     `json:"sync_count"`
	LastStatus      string    `json:"last_status"`
	LastError       string    `json:"last_error,omitempty"`
}

type Syncer struct {
	cfg        *config.Config
	client     *TelegramClient
	mu         sync.Mutex
	stats      SyncerStats
	workingDir string
}

func NewSyncer(cfg *config.Config) *Syncer {
	var tgClient *TelegramClient
	if cfg.IsTelegramConfigured() {
		tgClient = NewTelegramClient(cfg)
	}

	return &Syncer{
		cfg:        cfg,
		client:     tgClient,
		workingDir: "/tmp/yuki_mongo_vault",
		stats: SyncerStats{
			LastStatus: "initialized",
		},
	}
}

func (s *Syncer) GetStats() SyncerStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

// Restore checks the Telegram Storage channel and restores the latest snapshot
func (s *Syncer) Restore(port int) error {
	if s.client == nil || !s.cfg.IsTelegramConfigured() {
		log.Println("[Vault] Telegram credentials not configured. Skipping restore.")
		return nil
	}

	log.Println("[Vault] 📥 Checking Telegram Vault for latest database snapshot...")
	fileID, size, err := s.client.GetPinnedDocument(s.cfg.ChannelID)
	if err != nil {
		log.Printf("[Vault] No pinned snapshot found in channel: %v. Starting fresh.", err)
		return nil
	}

	log.Printf("[Vault] Found pinned snapshot (file_id: %s, size: %d bytes). Downloading...", fileID, size)
	_ = os.MkdirAll(s.workingDir, 0755)
	archivePath := fmt.Sprintf("%s/restore_snapshot.tar.gz", s.workingDir)
	defer os.Remove(archivePath)

	if err := s.client.DownloadFile(fileID, archivePath, s.cfg.ChannelID); err != nil {
		return fmt.Errorf("failed to download snapshot: %w", err)
	}

	extractDir := fmt.Sprintf("%s/restore_extracted", s.workingDir)
	_ = os.RemoveAll(extractDir)
	defer os.RemoveAll(extractDir)

	log.Println("[Vault] Extracting archive...")
	if err := ExtractTarGz(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	log.Println("[Vault] Restoring MongoDB dump into local instance...")
	if err := RestoreMongoDump(extractDir, port); err != nil {
		return fmt.Errorf("mongorestore error: %w", err)
	}

	s.mu.Lock()
	s.stats.LastStatus = "restored"
	s.stats.LastSyncTime = time.Now()
	s.mu.Unlock()

	log.Println("[Vault] ✅ [RESTORE COMPLETE] MongoDB successfully restored from Telegram Vault!")
	return nil
}

// Backup takes mongodump, compresses it, and uploads to Telegram channel
func (s *Syncer) Backup(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil || !s.cfg.IsTelegramConfigured() {
		return fmt.Errorf("telegram client not configured")
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	log.Printf("[Vault] Generating MongoDB dump snapshot at %s...", timestamp)

	_ = os.MkdirAll(s.workingDir, 0755)
	dumpDir := fmt.Sprintf("%s/mongodump", s.workingDir)
	archivePath := fmt.Sprintf("%s/yuki_mongo_backup.tar.gz", s.workingDir)
	defer os.RemoveAll(dumpDir)
	defer os.Remove(archivePath)

	if err := CreateMongoDump(dumpDir, port); err != nil {
		s.stats.LastStatus = "dump_failed"
		s.stats.LastError = err.Error()
		return fmt.Errorf("dump failed: %w", err)
	}

	// Check if any database was dumped
	entries, err := os.ReadDir(dumpDir)
	if err != nil || len(entries) == 0 {
		log.Println("[Vault] MongoDB has no user databases yet. Skipping empty dump upload.")
		return nil
	}

	log.Println("[Vault] Compressing mongodump into GZIP archive...")
	if err := CompressToTarGz(dumpDir, archivePath); err != nil {
		s.stats.LastStatus = "compression_failed"
		s.stats.LastError = err.Error()
		return fmt.Errorf("compression failed: %w", err)
	}

	fi, err := os.Stat(archivePath)
	if err != nil {
		return err
	}
	archiveSize := fi.Size()

	caption := fmt.Sprintf(
		"🛡️ **YukiMongo Vault Snapshot**\n📅 `%s`\n📦 Size: `%.2f KB`\n⚡ Engine: `Official MongoDB 6.0`\n🏷️ #YUKIMONGO_BACKUP",
		timestamp, float64(archiveSize)/1024.0,
	)

	log.Printf("[Vault] Uploading snapshot (%.2f KB) to Telegram channel %d...", float64(archiveSize)/1024.0, s.cfg.ChannelID)
	msgID, err := s.client.UploadDocument(s.cfg.ChannelID, archivePath, caption)
	if err != nil {
		s.stats.LastStatus = "upload_failed"
		s.stats.LastError = err.Error()
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("[Vault] ✅ [BACKUP SUCCESS] Uploaded MsgID %d. Pinning message...", msgID)
	_ = s.client.PinMessage(s.cfg.ChannelID, msgID)

	s.stats.LastSyncTime = time.Now()
	s.stats.LastArchiveSize = archiveSize
	s.stats.SyncCount++
	s.stats.LastStatus = "synced"
	s.stats.LastError = ""

	return nil
}

// StartSyncLoop runs periodic backup in a background goroutine
func (s *Syncer) StartSyncLoop(ctx context.Context, port int, intervalMin int) {
	if s.client == nil || !s.cfg.IsTelegramConfigured() {
		return
	}

	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	go func() {
		defer ticker.Stop()
		log.Printf("[Vault] Periodic auto-sync loop active (every %d minutes)", intervalMin)
		for {
			select {
			case <-ticker.C:
				if err := s.Backup(port); err != nil {
					log.Printf("[Vault] Periodic backup warning: %v", err)
				}
			case <-ctx.Done():
				log.Println("[Vault] Background sync loop exiting.")
				return
			}
		}
	}()
}
