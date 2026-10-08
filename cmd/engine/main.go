package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudeepbots/yuki-mongo-render/internal/config"
	"github.com/sudeepbots/yuki-mongo-render/internal/mongo"
	"github.com/sudeepbots/yuki-mongo-render/internal/server"
	"github.com/sudeepbots/yuki-mongo-render/internal/tunnel"
	"github.com/sudeepbots/yuki-mongo-render/internal/vault"
)

func main() {
	log.Println("=================================================================")
	log.Println("🍃 Yuki-Mongo-Render: Enterprise Go Orchestration Engine v1.0")
	log.Println("=================================================================")

	cfg := config.LoadConfig()

	// 1. Initialize Subsystems
	mongoSupervisor := mongo.NewSupervisor(cfg)
	tunnelSupervisor := tunnel.NewTunnel(cfg)
	vaultSyncer := vault.NewSyncer(cfg)
	httpServer := server.NewServer(cfg, mongoSupervisor, tunnelSupervisor, vaultSyncer)

	// 2. Start Official MongoDB
	if err := mongoSupervisor.Start(); err != nil {
		log.Fatalf("[FATAL] Could not launch MongoDB engine: %v", err)
	}

	// 3. Wait for MongoDB to accept connections
	if err := mongoSupervisor.WaitForReady(30 * time.Second); err != nil {
		log.Fatalf("[FATAL] MongoDB failed readiness check: %v", err)
	}

	// 4. Restore Latest Snapshot from Telegram Vault
	if cfg.IsTelegramConfigured() {
		if err := vaultSyncer.Restore(cfg.MongoPort); err != nil {
			log.Printf("[WARN] Vault restore notice: %v", err)
		}
	}

	// 5. Start Background Ticker Sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.IsTelegramConfigured() {
		vaultSyncer.StartSyncLoop(ctx, cfg.MongoPort, cfg.SyncIntervalMin)
	}

	// 6. Start Cloudflare Tunnel
	if cfg.IsTunnelConfigured() {
		if err := tunnelSupervisor.Start(); err != nil {
			log.Printf("[WARN] Cloudflare tunnel launch warning: %v", err)
		}
	}

	// 7. Graceful Shutdown & Signal Handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[Engine] Received termination signal (%v). Initiating graceful shutdown...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer shutdownCancel()

		// Emergency final backup before termination
		if cfg.IsTelegramConfigured() {
			log.Println("[Engine] 🛡️ Taking emergency final snapshot for Telegram Vault...")
			if err := vaultSyncer.Backup(cfg.MongoPort); err != nil {
				log.Printf("[Engine] Emergency backup notice: %v", err)
			}
		}

		cancel()
		_ = httpServer.Stop(shutdownCtx)
		_ = tunnelSupervisor.Stop(shutdownCtx)
		_ = mongoSupervisor.Stop(shutdownCtx)

		log.Println("[Engine] Clean shutdown finished. Exiting.")
		os.Exit(0)
	}()

	// 8. Run Foreground HTTP Server for Render
	if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[FATAL] HTTP Server terminated: %v", err)
	}
}
