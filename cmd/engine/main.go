package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
	"github.com/sudeepbots/YUKI-MONGODB/internal/mongo"
	"github.com/sudeepbots/YUKI-MONGODB/internal/server"
	"github.com/sudeepbots/YUKI-MONGODB/internal/tunnel"
	"github.com/sudeepbots/YUKI-MONGODB/internal/vault"
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

	// 4. Configure Authentication Credentials
	_ = mongoSupervisor.ConfigureAuth()

	// 5. Restore Latest Snapshot from Telegram Vault
	if cfg.IsTelegramConfigured() {
		if err := vaultSyncer.Restore(cfg.MongoPort); err != nil {
			log.Printf("[WARN] Vault restore notice: %v", err)
		}
	}

	// 6. Start Background Ticker Sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.IsTelegramConfigured() {
		vaultSyncer.StartSyncLoop(ctx, cfg.MongoPort, cfg.SyncIntervalMin)
	}

	// 7. Start Tunnel (Cloudflare Zero Trust or Auto-TCP Relay)
	if err := tunnelSupervisor.Start(); err != nil {
		log.Printf("[WARN] Tunnel launch warning: %v", err)
	}

	// 8. Display MongoDB Connection Banner in Logs
	go func() {
		time.Sleep(2 * time.Second)
		host, port := tunnelSupervisor.GetEndpoint()
		uri := mongoSupervisor.GetConnectionString(host, port)
		log.Println("=================================================================")
		log.Println("🍃 YUKI-MONGODB ONLINE & READY FOR BOT CONNECTIONS!")
		log.Printf("👉 Real Mongo URI : %s", uri)
		log.Printf("👉 Internal Socket: 127.0.0.1:%d", cfg.MongoPort)
		log.Printf("👉 Render Health  : 0.0.0.0:%d/health", cfg.Port)
		log.Println("=================================================================")
	}()

	// 9. Graceful Shutdown & Signal Handling
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

	// 10. Run Foreground HTTP Server for Render
	if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[FATAL] HTTP Server terminated: %v", err)
	}
}
