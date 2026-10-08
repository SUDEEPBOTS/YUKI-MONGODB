/**
 * =================================================================================================
 *   __     __ _    _  _  __ _____     __  __  ____   _   _   ____   ____   ____   ____  
 *   \ \   / /| |  | || |/ /|_   _|   |  \/  |/ __ \ | \ | | / ___| / __ \ |  _ \ | __ ) 
 *    \ \ / / | |  | || ' /   | |     | |\/| || |  | ||  \| || |  _ | |  | || | | ||  _ \ 
 *     \ V /  | |__| || . \  _| |_    | |  | || |__| || |\  || |_| || |__| || |_| || |_) |
 *      \_/    \____/ |_|\_\|_____|   |_|  |_| \____/ |_| \_| \____| \____/ |____/ |____/  
 *
 * =================================================================================================
 * Project       : YUKI-MONGODB
 * Official Repo : https://github.com/SUDEEPBOTS/YUKI-MONGODB
 * Description   : Enterprise Official MongoDB 6.0 Engine for Render Free Tier & Cloud Platforms
 * Architecture  : Pure Go Orchestrator, MTProto Cloud Vault, Cloudflare Zero Trust Argo Tunnel
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 * Author / Dev  : Sudeep & Yuki Network Engineering Team
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 *
 * Licensed under the MIT License (the "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at:
 *
 *     https://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 * =================================================================================================
 */

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
	log.Println("🍃 YUKI-MONGODB: Enterprise Orchestration Engine v1.0")
	log.Println("=================================================================")

	cfg := config.LoadConfig()

	mongoSupervisor := mongo.NewSupervisor(cfg)
	tunnelSupervisor := tunnel.NewTunnel(cfg)
	vaultSyncer := vault.NewSyncer(cfg)
	httpServer := server.NewServer(cfg, mongoSupervisor, tunnelSupervisor, vaultSyncer)

	if err := mongoSupervisor.Start(); err != nil {
		log.Fatalf("[FATAL] Could not launch MongoDB engine: %v", err)
	}

	if err := mongoSupervisor.WaitForReady(30 * time.Second); err != nil {
		log.Fatalf("[FATAL] MongoDB failed readiness check: %v", err)
	}

	_ = mongoSupervisor.ConfigureAuth()

	if cfg.IsTelegramConfigured() {
		if err := vaultSyncer.Restore(cfg.MongoPort); err != nil {
			log.Printf("[WARN] Vault restore notice: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cfg.IsTelegramConfigured() {
		vaultSyncer.StartSyncLoop(ctx, cfg.MongoPort, cfg.SyncIntervalMin)
	}

	if err := tunnelSupervisor.Start(); err != nil {
		log.Printf("[WARN] Tunnel launch warning: %v", err)
	}

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[Engine] Received termination signal (%v). Initiating graceful shutdown...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer shutdownCancel()

		if cfg.IsTelegramConfigured() {
			log.Println("[Engine] 🛡️ Taking emergency final snapshot for Cloud Vault...")
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

	if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[FATAL] HTTP Server terminated: %v", err)
	}
}
