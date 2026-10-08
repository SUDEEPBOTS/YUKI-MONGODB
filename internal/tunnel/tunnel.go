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
 * Description   : Cloudflare Zero Trust Argo Tunnel & Automated TCP Relay Manager
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

package tunnel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"time"

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

// Start launches Multi-Tunnel Orchestrator for MongoDB
func (t *Tunnel) Start() error {
	log.Println("[Tunnel] ⚡ Launching Multi-Tunnel Orchestrator for MongoDB...")
	t.running = true

	// 1. Launch Cloudflare Tunnel daemon if configured (for VPS forwarder & Zero Trust)
	if t.cfg.IsTunnelConfigured() {
		go func() {
			args := []string{
				"tunnel",
				"--no-autoupdate",
				"run",
				"--token", t.cfg.TunnelToken,
			}
			cmd := exec.Command("cloudflared", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Println("[Cloudflare Tunnel] ⚡ Starting cloudflared tunnel daemon...")
			if err := cmd.Run(); err != nil {
				log.Printf("[Cloudflare Tunnel] cloudflared exited: %v", err)
			}
		}()
	}

	// 2. Also launch Pinggy direct TCP relay
	go t.startAutoTCPRelay()

	return nil
}

func (t *Tunnel) startBoreTunnel() {
	boreBin, err := exec.LookPath("bore")
	if err != nil {
		log.Println("[Bore] bore binary not found, fallback to Pinggy...")
		return
	}

	for {
		args := []string{
			"local",
			"--to", "bore.pub",
			"--port", strconv.Itoa(t.cfg.StaticPort),
			strconv.Itoa(t.cfg.MongoPort),
		}

		cmd := exec.Command(boreBin, args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("[Bore] Stderr pipe error: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("[Bore] Launch error: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(stderr)
		re := regexp.MustCompile(`listening at ([a-zA-Z0-9.-]+):(\d+)`)
		re2 := regexp.MustCompile(`remote_port=(\d+)`)
		for scanner.Scan() {
			line := scanner.Text()
			if match := re.FindStringSubmatch(line); len(match) == 3 {
				host := match[1]
				port, _ := strconv.Atoi(match[2])
				t.updateEndpoint(host, port, "LIFETIME STATIC TCP")
			} else if match := re2.FindStringSubmatch(line); len(match) == 2 {
				port, _ := strconv.Atoi(match[1])
				t.updateEndpoint("bore.pub", port, "LIFETIME STATIC TCP")
			}
		}

		_ = cmd.Wait()
		log.Println("[Bore] ⚠️ Connection closed. Reconnecting immediately with static port...")
		time.Sleep(2 * time.Second)
	}
}

func (t *Tunnel) updateEndpoint(host string, port int, label string) {
	if host == "" || port <= 0 {
		return
	}
	t.publicHost = host
	t.publicPort = port

	log.Println("=================================================================")
	log.Printf("🍃 [Tunnel] %s ACTIVE: %s:%d", label, host, port)
	if t.cfg.HasMongoAuth() {
		log.Printf("👉 Real Mongo URI : mongodb://%s:%s@%s:%d/?authSource=admin", t.cfg.MongoUser, t.cfg.MongoPass, host, port)
	} else {
		log.Printf("👉 Real Mongo URI : mongodb://%s:%d", host, port)
	}
	log.Println("=================================================================")
}

func (t *Tunnel) startAutoTCPRelay() error {
	t.isFallback = true

	for {
		sshArgs := []string{
			"-p", "443",
			"-o", "StrictHostKeyChecking=no",
			"-o", "ServerAliveInterval=30",
			"-o", "ServerAliveCountMax=3",
			"-o", "ExitOnForwardFailure=yes",
			"-R", fmt.Sprintf("0:localhost:%d", t.cfg.MongoPort),
			"tcp@a.pinggy.io",
		}

		cmd := exec.Command("ssh", sshArgs...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("[Auto-TCP] SSH pipe error: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		cmd.Stderr = cmd.Stdout
		t.cmd = cmd

		if err := cmd.Start(); err != nil {
			log.Printf("[Auto-TCP] SSH launch error: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(stdout)
		re := regexp.MustCompile(`tcp://([a-zA-Z0-9.-]+):(\d+)`)
		re2 := regexp.MustCompile(`([a-zA-Z0-9.-]+(?:pinggy|a\.pinggy)[a-zA-Z0-9.-]*):(\d+)`)
		for scanner.Scan() {
			line := scanner.Text()
			var host string
			var port int
			if match := re.FindStringSubmatch(line); len(match) == 3 {
				host = match[1]
				port, _ = strconv.Atoi(match[2])
			} else if match := re2.FindStringSubmatch(line); len(match) == 3 {
				host = match[1]
				port, _ = strconv.Atoi(match[2])
			}
			if host != "" && port > 0 && t.publicHost != "bore.pub" {
				t.updateEndpoint(host, port, "Auto-TCP Pinggy Relay")
			}
		}

		_ = cmd.Wait()
		log.Println("[Auto-TCP] ⚠️ SSH connection closed. Reconnecting immediately...")
		time.Sleep(2 * time.Second)
	}
}

func (t *Tunnel) syncCloudflareDNS(targetHost string) {
	if t.cfg.CFKey == "" || t.cfg.CFEmail == "" || t.cfg.CFZoneID == "" || t.cfg.Domain == "" {
		return
	}

	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		listURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", t.cfg.CFZoneID, t.cfg.Domain)
		req, err := http.NewRequest("GET", listURL, nil)
		if err != nil {
			return
		}
		req.Header.Set("X-Auth-Key", t.cfg.CFKey)
		req.Header.Set("X-Auth-Email", t.cfg.CFEmail)

		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var listResp struct {
			Success bool `json:"success"`
			Result  []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil || len(listResp.Result) == 0 {
			return
		}

		recID := listResp.Result[0].ID
		if listResp.Result[0].Content == targetHost {
			log.Printf("[Cloudflare DNS] ✅ %s is already synced to %s", t.cfg.Domain, targetHost)
			return
		}

		putURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", t.cfg.CFZoneID, recID)
		payload := map[string]interface{}{
			"type":    "CNAME",
			"name":    t.cfg.Domain,
			"content": targetHost,
			"proxied": false,
			"ttl":     60,
		}
		b, _ := json.Marshal(payload)
		putReq, err := http.NewRequest("PUT", putURL, bytes.NewBuffer(b))
		if err != nil {
			return
		}
		putReq.Header.Set("X-Auth-Key", t.cfg.CFKey)
		putReq.Header.Set("X-Auth-Email", t.cfg.CFEmail)
		putReq.Header.Set("Content-Type", "application/json")

		putResp, err := client.Do(putReq)
		if err == nil {
			_ = putResp.Body.Close()
			log.Printf("[Cloudflare DNS] 🌐 Auto-updated %s -> %s (Proxied: false, TTL: 60s)", t.cfg.Domain, targetHost)
		}
	}()
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
