package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"

	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
)

type Tunnel struct {
	cfg        *config.Config
	cmd        *exec.Cmd
	running    bool
	isFallback bool
	publicHost string
	publicPort int
}

func NewTunnel(cfg *config.Config) *Tunnel {
	return &Tunnel{
		cfg:        cfg,
		publicHost: cfg.Domain,
		publicPort: cfg.MongoPort,
	}
}

// Start launches Cloudflare Zero Trust tunnel or Auto-TCP relay fallback
func (t *Tunnel) Start() error {
	if t.cfg.IsTunnelConfigured() {
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
		t.publicHost = t.cfg.Domain
		t.publicPort = t.cfg.MongoPort
		return nil
	}

	// Fallback Mode: Auto Free TCP Relay via Pinggy (No token/domain required)
	log.Println("[Auto-TCP] ⚠️ No TUNNEL_TOKEN provided. Launching Zero-Config Free TCP Tunnel...")
	return t.startAutoTCPRelay()
}

func (t *Tunnel) startAutoTCPRelay() error {
	sshArgs := []string{
		"-p", "443",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ServerAliveInterval=30",
		"-R", fmt.Sprintf("0:localhost:%d", t.cfg.MongoPort),
		"tcp@a.pinggy.io",
	}

	t.cmd = exec.Command("ssh", sshArgs...)
	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		log.Printf("[Auto-TCP] SSH pipe error: %v. Running in local-only mode.", err)
		return nil
	}
	t.cmd.Stderr = t.cmd.Stdout

	if err := t.cmd.Start(); err != nil {
		log.Printf("[Auto-TCP] SSH relay launch warning: %v. Local MongoDB active.", err)
		return nil
	}

	t.running = true
	t.isFallback = true

	// Parse public TCP port from pinggy output in background
	go func() {
		scanner := bufio.NewScanner(stdout)
		re := regexp.MustCompile(`(a\.pinggy\.io|tcp\.pinggy\.io):(\d+)`)
		for scanner.Scan() {
			line := scanner.Text()
			if match := re.FindStringSubmatch(line); len(match) == 3 {
				t.publicHost = match[1]
				t.publicPort, _ = strconv.Atoi(match[2])
				log.Printf("[Auto-TCP] 🌟 Live Free TCP Relay Allocated: %s:%d", t.publicHost, t.publicPort)
				break
			}
		}
	}()

	return nil
}

func (t *Tunnel) GetEndpoint() (string, int) {
	return t.publicHost, t.publicPort
}

func (t *Tunnel) IsRunning() bool {
	return t.running && t.cmd != nil && t.cmd.Process != nil
}

func (t *Tunnel) Stop(ctx context.Context) error {
	if !t.running || t.cmd == nil || t.cmd.Process == nil {
		return nil
	}

	log.Println("[Tunnel] Stopping tunnel daemon...")
	_ = t.cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case <-done:
		log.Println("[Tunnel] Tunnel stopped cleanly.")
		return nil
	case <-ctx.Done():
		_ = t.cmd.Process.Kill()
		return ctx.Err()
	}
}
