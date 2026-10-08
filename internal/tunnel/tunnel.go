package tunnel

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/sudeepbots/yuki-mongo-render/internal/config"
)

type Tunnel struct {
	cfg     *config.Config
	cmd     *exec.Cmd
	running bool
}

func NewTunnel(cfg *config.Config) *Tunnel {
	return &Tunnel{
		cfg: cfg,
	}
}

// Start launches cloudflared tunnel daemon
func (t *Tunnel) Start() error {
	if !t.cfg.IsTunnelConfigured() {
		log.Println("[Cloudflare Tunnel] ⚠️ TUNNEL_TOKEN not provided. Skipping tunnel launch.")
		return nil
	}

	args := []string{
		"tunnel",
		"--no-autoupdate",
		"run",
		"--token", t.cfg.TunnelToken,
	}

	t.cmd = exec.Command("cloudflared", args...)
	t.cmd.Stdout = os.Stdout
	t.cmd.Stderr = os.Stderr

	log.Println("[Cloudflare Tunnel] ⚡ Starting cloudflared tunnel daemon...")
	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %w", err)
	}

	t.running = true
	return nil
}

func (t *Tunnel) IsRunning() bool {
	return t.running && t.cmd != nil && t.cmd.Process != nil
}

func (t *Tunnel) Stop(ctx context.Context) error {
	if !t.running || t.cmd == nil || t.cmd.Process == nil {
		return nil
	}

	log.Println("[Cloudflare Tunnel] Stopping cloudflared daemon...")
	_ = t.cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case <-done:
		log.Println("[Cloudflare Tunnel] cloudflared stopped.")
		return nil
	case <-ctx.Done():
		_ = t.cmd.Process.Kill()
		return ctx.Err()
	}
}
