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
 * Description   : Official MongoDB 6.0 Process Supervisor & Lifecycle Engine
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

package mongo

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
)

type Supervisor struct {
	cfg     *config.Config
	cmd     *exec.Cmd
	started bool
}

func NewSupervisor(cfg *config.Config) *Supervisor {
	return &Supervisor{
		cfg: cfg,
	}
}

// Start spawns the official mongod process with memory limits tuned for Render
func (s *Supervisor) Start() error {
	_ = os.MkdirAll(s.cfg.DataDir, 0755)

	args := []string{
		"--bind_ip_all",
		"--port", fmt.Sprintf("%d", s.cfg.MongoPort),
		"--dbpath", s.cfg.DataDir,
		"--wiredTigerCacheSizeGB", fmt.Sprintf("%.2f", s.cfg.CacheSizeGB),
	}

	s.cmd = exec.Command("mongod", args...)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	log.Printf("[MongoDB] Launching mongod on port %d (WiredTiger capped at %.2f GB)...", s.cfg.MongoPort, s.cfg.CacheSizeGB)
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mongod: %w", err)
	}

	s.started = true
	return nil
}

// WaitForReady polls the MongoDB TCP port until it accepts connections
func (s *Supervisor) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.MongoPort)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			log.Printf("[MongoDB] ✅ MongoDB is online and accepting connections at %s!", addr)
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for MongoDB socket at %s", addr)
}

// ConfigureAuth sets up root administrative credentials in MongoDB
func (s *Supervisor) ConfigureAuth() error {
	if !s.cfg.HasMongoAuth() {
		log.Println("[MongoDB] No MONGO_USER or MONGO_PASS provided. Running without authentication.")
		return nil
	}

	jsScript := fmt.Sprintf(`
try {
  db.getSiblingDB("admin").createUser({
    user: "%s",
    pwd: "%s",
    roles: [ { role: "root", db: "admin" } ]
  });
  print("USER_CREATED");
} catch(e) {
  if (e.code === 51003 || ("" + e).indexOf("already exists") !== -1) {
    db.getSiblingDB("admin").changeUserPassword("%s", "%s");
    print("PASSWORD_UPDATED");
  } else {
    print("NOTICE: " + e);
  }
}
`, s.cfg.MongoUser, s.cfg.MongoPass, s.cfg.MongoUser, s.cfg.MongoPass)

	cmd := exec.Command("mongosh", "--port", fmt.Sprintf("%d", s.cfg.MongoPort), "--eval", jsScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[WARN] [MongoDB Auth] mongosh notice: %v, out: %s", err, string(out))
		return nil
	}

	log.Printf("[MongoDB] 🔐 Root admin credentials enabled for user: '%s'", s.cfg.MongoUser)
	return nil
}

// GetConnectionString formats standard MongoDB connection string
func (s *Supervisor) GetConnectionString(host string, port int) string {
	if s.cfg.HasMongoAuth() {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin", s.cfg.MongoUser, s.cfg.MongoPass, host, port)
	}
	return fmt.Sprintf("mongodb://%s:%d", host, port)
}

// IsAlive checks if MongoDB TCP socket responds
func (s *Supervisor) IsAlive() bool {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.MongoPort)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Stop sends SIGTERM to mongod for clean shutdown
func (s *Supervisor) Stop(ctx context.Context) error {
	if s.cmd == nil || s.cmd.Process == nil || !s.started {
		return nil
	}

	log.Println("[MongoDB] Sending SIGTERM to mongod for graceful shutdown...")
	_ = s.cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
		log.Println("[MongoDB] mongod stopped cleanly.")
		return nil
	case <-ctx.Done():
		log.Println("[MongoDB] Graceful shutdown timeout exceeded, killing mongod...")
		_ = s.cmd.Process.Kill()
		return ctx.Err()
	}
}
