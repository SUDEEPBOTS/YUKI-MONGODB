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

	"github.com/sudeepbots/yuki-mongo-render/internal/config"
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
