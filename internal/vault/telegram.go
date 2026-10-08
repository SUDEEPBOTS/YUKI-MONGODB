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
 * Description   : Telegram Native MTProto & Bot API Cloud Vault Engine
 * Maintainer    : SUDEEPBOTS <https://github.com/SUDEEPBOTS>
 *
 * Copyright (c) 2026 SUDEEPBOTS. All rights reserved.
 * Licensed under the MIT License (https://opensource.org/licenses/MIT)
 * =================================================================================================
 */

package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/sudeepbots/YUKI-MONGODB/internal/config"
)

var dcList = map[int]string{
	1: "149.154.175.58:443",
	2: "149.154.167.50:443",
	3: "149.154.175.100:443",
	4: "149.154.167.91:443",
	5: "91.108.56.151:443",
}

type TelegramClient struct {
	cfg        *config.Config
	isMTProto  bool
	mtClient   *telegram.Client
	httpClient *http.Client
	mu         sync.Mutex
}

func decodePyrogramSession(encodedString string) (*telegram.Session, error) {
	encodedString = strings.TrimSpace(encodedString)
	if encodedString == "" {
		return nil, errors.New("empty session string provided")
	}

	for len(encodedString)%4 != 0 {
		encodedString += "="
	}

	packedData, err := base64.URLEncoding.DecodeString(encodedString)
	if err != nil {
		packedData, err = base64.StdEncoding.DecodeString(encodedString)
		if err != nil {
			return nil, fmt.Errorf("base64 session decode error: %w", err)
		}
	}

	const (
		dcIDSize     = 1
		apiIDSize    = 4
		testModeSize = 1
		authKeySize  = 256
		userIDSize   = 8
		isBotSize    = 1
	)

	expectedSize := dcIDSize + apiIDSize + testModeSize + authKeySize + userIDSize + isBotSize
	if len(packedData) != expectedSize {
		return nil, fmt.Errorf("invalid pyrogram session length (got %d, expected %d)", len(packedData), expectedSize)
	}

	appID := int32(uint32(packedData[1])<<24 | uint32(packedData[2])<<16 | uint32(packedData[3])<<8 | uint32(packedData[4]))
	dcID := int(packedData[0])
	ip := dcList[dcID]
	if ip == "" {
		ip = "149.154.167.91:443"
	}

	authKey := make([]byte, authKeySize)
	copy(authKey, packedData[6:6+authKeySize])

	return &telegram.Session{
		Hostname: ip,
		AppID:    appID,
		Key:      authKey,
	}, nil
}

func NewTelegramClient(cfg *config.Config) *TelegramClient {
	tc := &TelegramClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}

	if cfg.SessionString != "" {
		sess, err := decodePyrogramSession(cfg.SessionString)
		if err == nil {
			appID := sess.AppID
			if appID == 0 {
				appID = cfg.AppID
			}

			client, err := telegram.NewClient(telegram.ClientConfig{
				AppID:         appID,
				AppHash:       cfg.AppHash,
				StringSession: sess.Encode(),
				NoUpdates:     true,
				LogLevel:      telegram.LogWarn,
			})

			if err == nil {
				if err := client.Connect(); err == nil {
					tc.isMTProto = true
					tc.mtClient = client
					log.Printf("[MTProto Worker] ⚡ Connected natively to Telegram DC %s via MTProto session!", sess.Hostname)
					return tc
				} else {
					log.Printf("[WARN] [MTProto Worker] Connection failed: %v", err)
				}
			} else {
				log.Printf("[WARN] [MTProto Worker] Client creation failed: %v", err)
			}
		} else {
			log.Printf("[WARN] [MTProto Worker] Failed to decode SESSION_STRING: %v", err)
		}
	}

	log.Println("[Vault] Operating in standard Telegram Bot API mode.")
	return tc
}

func (t *TelegramClient) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", t.cfg.BotToken, method)
}

// GetPinnedDocument retrieves the file ID or message ID from the pinned message
func (t *TelegramClient) GetPinnedDocument(channelID int64) (string, int64, error) {
	if t.isMTProto && t.mtClient != nil {
		msg, err := t.mtClient.GetPinnedMessage(channelID)
		if err == nil && msg != nil && msg.Media() != nil {
			return fmt.Sprintf("msg:%d", msg.ID), int64(msg.ID), nil
		}
	}

	if t.cfg.BotToken == "" {
		return "", 0, fmt.Errorf("no bot token or MTProto connection")
	}

	url := fmt.Sprintf("%s?chat_id=%d", t.apiURL("getChat"), channelID)
	resp, err := t.httpClient.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var res struct {
		OK     bool `json:"ok"`
		Result struct {
			PinnedMessage struct {
				Document struct {
					FileID   string `json:"file_id"`
					FileSize int64  `json:"file_size"`
				} `json:"document"`
			} `json:"pinned_message"`
		} `json:"result"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", 0, err
	}
	if !res.OK {
		return "", 0, fmt.Errorf("bot API error: %s", res.Description)
	}

	fileID := res.Result.PinnedMessage.Document.FileID
	if fileID == "" {
		return "", 0, fmt.Errorf("no pinned document")
	}

	return fileID, res.Result.PinnedMessage.Document.FileSize, nil
}

// DownloadFile downloads a document snapshot
func (t *TelegramClient) DownloadFile(fileID string, destPath string, channelID int64) error {
	if t.isMTProto && t.mtClient != nil && strings.HasPrefix(fileID, "msg:") {
		msgIDStr := strings.TrimPrefix(fileID, "msg:")
		msgID, _ := strconv.Atoi(msgIDStr)
		msg, err := t.mtClient.GetMessageByID(channelID, int32(msgID))
		if err != nil {
			return fmt.Errorf("failed to get message %d: %w", msgID, err)
		}
		media := msg.Media()
		if media == nil {
			return fmt.Errorf("message %d has no media", msgID)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = t.mtClient.DownloadMedia(media, &telegram.DownloadOptions{
			Buffer:  outFile,
			Threads: 4,
		})
		return err
	}

	getURL := fmt.Sprintf("%s?file_id=%s", t.apiURL("getFile"), fileID)
	resp, err := t.httpClient.Get(getURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.cfg.BotToken, res.Result.FilePath)
	dlResp, err := t.httpClient.Get(downloadURL)
	if err != nil {
		return err
	}
	defer dlResp.Body.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, dlResp.Body)
	return err
}

// UploadDocument uploads snapshot archive to Telegram channel
func (t *TelegramClient) UploadDocument(channelID int64, filePath, caption string) (int64, error) {
	if t.isMTProto && t.mtClient != nil {
		log.Printf("[MTProto Worker] Uploading archive %s directly via MTProto socket...", filepath.Base(filePath))
		sent, err := t.mtClient.SendMedia(channelID, filePath, &telegram.MediaOptions{
			Caption: caption,
		})
		if err == nil && sent != nil && sent.ID > 0 {
			return int64(sent.ID), nil
		}
		log.Printf("[WARN] MTProto upload notice: %v, falling back to HTTP...", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", strconv.FormatInt(channelID, 10))
	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("parse_mode", "Markdown")

	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return 0, err
	}
	_ = writer.Close()

	req, err := http.NewRequest("POST", t.apiURL("sendDocument"), body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var res struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	if !res.OK {
		return 0, fmt.Errorf("upload failed: %s", res.Description)
	}

	return res.Result.MessageID, nil
}

// PinMessage pins the uploaded document message in the channel
func (t *TelegramClient) PinMessage(channelID int64, msgID int64) error {
	if t.isMTProto && t.mtClient != nil {
		_, _ = t.mtClient.PinMessage(channelID, int32(msgID))
		return nil
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"chat_id":              channelID,
		"message_id":           msgID,
		"disable_notification": true,
	})
	req, err := http.NewRequest("POST", t.apiURL("pinChatMessage"), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	return nil
}
