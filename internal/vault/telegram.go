package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type TelegramClient struct {
	token      string
	httpClient *http.Client
}

func NewTelegramClient(token string) *TelegramClient {
	return &TelegramClient{
		token: token,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (t *TelegramClient) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", t.token, method)
}

// GetPinnedDocument retrieves the file ID from the pinned message of the storage channel
func (t *TelegramClient) GetPinnedDocument(channelID int64) (string, int64, error) {
	url := fmt.Sprintf("%s?chat_id=%d", t.apiURL("getChat"), channelID)
	resp, err := t.httpClient.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("telegram getChat error: %w", err)
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
		return "", 0, fmt.Errorf("telegram API error: %s", res.Description)
	}

	fileID := res.Result.PinnedMessage.Document.FileID
	fileSize := res.Result.PinnedMessage.Document.FileSize
	if fileID == "" {
		return "", 0, fmt.Errorf("no document found in channel pinned message")
	}

	return fileID, fileSize, nil
}

// DownloadFile downloads a document by its file_id to destination path
func (t *TelegramClient) DownloadFile(fileID, destPath string) error {
	// Step 1: getFile
	getURL := fmt.Sprintf("%s?file_id=%s", t.apiURL("getFile"), fileID)
	resp, err := t.httpClient.Get(getURL)
	if err != nil {
		return fmt.Errorf("getFile error: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if !res.OK || res.Result.FilePath == "" {
		return fmt.Errorf("failed to resolve file path: %s", res.Description)
	}

	// Step 2: Download stream
	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.token, res.Result.FilePath)
	dlResp, err := t.httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", dlResp.StatusCode)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, dlResp.Body)
	return err
}

// UploadDocument uploads an archive document to the Telegram channel
func (t *TelegramClient) UploadDocument(channelID int64, filePath, caption string) (int64, error) {
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
		return 0, fmt.Errorf("sendDocument error: %w", err)
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
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("[Vault] Pinned latest backup message %d in channel %d", msgID, channelID)
	return nil
}
