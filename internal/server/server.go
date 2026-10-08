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
 * Description   : HTTP Health, Keep-Alive, & Diagnostic Telemetry Server
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
	"github.com/sudeepbots/YUKI-MONGODB/internal/mongo"
	"github.com/sudeepbots/YUKI-MONGODB/internal/tunnel"
	"github.com/sudeepbots/YUKI-MONGODB/internal/vault"
)

type Server struct {
	cfg        *config.Config
	mongo      *mongo.Supervisor
	tunnel     *tunnel.Tunnel
	syncer     *vault.Syncer
	startTime  time.Time
	httpServer *http.Server
}

func NewServer(cfg *config.Config, m *mongo.Supervisor, t *tunnel.Tunnel, s *vault.Syncer) *Server {
	return &Server{
		cfg:       cfg,
		mongo:     m,
		tunnel:    t,
		syncer:    s,
		startTime: time.Now(),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleHealth)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ping", s.handleHealth)
	mux.HandleFunc("/backup", s.handleManualBackup)
	mux.HandleFunc("/stats", s.handleStats)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", s.cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[Server] 🚀 Render HTTP Health server listening on 0.0.0.0:%d", s.cfg.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed. Allowed methods: GET, HEAD",
		})
		return
	}

	mongoAlive := s.mongo.IsAlive()
	state := "healthy"
	statusCode := http.StatusOK
	if !mongoAlive {
		state = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if r.Method == http.MethodHead {
		return
	}

	resp := map[string]interface{}{
		"status":          state,
		"service":         "Yuki-MongoDB-Render (Go Engine)",
		"mongodb_port":    s.cfg.MongoPort,
		"mongodb_alive":   mongoAlive,
		"tunnel_running":  s.tunnel.IsRunning(),
		"storage_mode":    "local_ephemeral",
		"persistent":      false,
		"uptime_seconds":  int(time.Since(s.startTime).Seconds()),
		"server_time_utc": time.Now().UTC().Format(time.RFC3339),
	}

	if s.cfg.IsTelegramConfigured() {
		resp["storage_mode"] = "cloud_vault"
		resp["persistent"] = true
		resp["telegram_vault"] = true
	} else {
		resp["warning"] = "Ephemeral storage active. All data will be deleted when container restarts. Configure SESSION_STRING & CHANNEL_ID for persistent cloud vault."
		resp["telegram_vault"] = false
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleManualBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	go func() {
		if err := s.syncer.Backup(s.cfg.MongoPort); err != nil {
			log.Printf("[Server] Manual backup triggered error: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "triggered",
		"message": "MongoDB snapshot backup triggered in background",
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	storageMode := "local_ephemeral"
	if s.cfg.IsTelegramConfigured() {
		storageMode = "cloud_vault"
	}

	stats := map[string]interface{}{
		"service":      "Yuki-MongoDB-Render",
		"version":      "1.0.0-golang",
		"storage_mode": storageMode,
		"uptime":       time.Since(s.startTime).String(),
		"mongo": map[string]interface{}{
			"port":          s.cfg.MongoPort,
			"alive":         s.mongo.IsAlive(),
			"cache_size_gb": s.cfg.CacheSizeGB,
		},
		"tunnel": map[string]interface{}{
			"active": s.tunnel.IsRunning(),
		},
		"vault": s.syncer.GetStats(),
	}
	_ = json.NewEncoder(w).Encode(stats)
}
