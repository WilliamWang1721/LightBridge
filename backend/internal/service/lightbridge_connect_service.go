package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// LightBridgeConnectType represents the type of external service
type LightBridgeConnectType string

const (
	LBCTypeNewAPI LightBridgeConnectType = "new-api"
)

// LightBridgeConnectConfig represents the complete configuration
type LightBridgeConnectConfig struct {
	Type           LightBridgeConnectType `json:"type"`
	InstanceURL    string                 `json:"instance_url"`
	SystemToken    string                 `json:"system_token"` // Encrypted
	UserID         int                    `json:"user_id,omitempty"`
	Username       string                 `json:"username,omitempty"`
	Quota          *QuotaInfo             `json:"quota,omitempty"`
	Alert          *AlertConfig           `json:"alert,omitempty"`
	WebhookURL     string                 `json:"webhook_url,omitempty"`
	SyncInterval   int                    `json:"sync_interval"` // seconds
	LastVerifiedAt *time.Time             `json:"last_verified_at,omitempty"`
}

// QuotaInfo stores quota information
type QuotaInfo struct {
	Balance    int64      `json:"balance"` // in cents/fen
	Used       int64      `json:"used"`    // in cents/fen
	LastSyncAt *time.Time `json:"last_sync_at,omitempty"`
	Currency   string     `json:"currency"` // CNY, USD, etc.
}

// AlertConfig defines alert configuration
type AlertConfig struct {
	Enabled          bool     `json:"enabled"`
	Threshold        int64    `json:"threshold"` // in cents/fen
	Channels         []string `json:"channels"`  // email, webhook, dashboard
	AutoDisableOnLow bool     `json:"auto_disable_on_low"`
}

// NewAPIUserResponse represents New API /api/user/self response
type NewAPIUserResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID          int    `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        int    `json:"role"`
		Status      int    `json:"status"`
		Group       string `json:"group"`
		Quota       int    `json:"quota"`      // total quota in cents
		UsedQuota   int    `json:"used_quota"` // used quota in cents
	} `json:"data"`
}

// LightBridgeConnectService handles LBC operations
type LightBridgeConnectService struct {
	encryptionKey []byte // 32-byte key for AES-256
}

// NewLightBridgeConnectService creates a new service instance
func NewLightBridgeConnectService(encryptionKey string) *LightBridgeConnectService {
	// Ensure 32-byte key for AES-256
	key := []byte(encryptionKey)
	if len(key) < 32 {
		// Pad with zeros if too short
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		// Truncate if too long
		key = key[:32]
	}

	return &LightBridgeConnectService{
		encryptionKey: key,
	}
}

// EncryptToken encrypts a system token using AES-256-GCM
func (s *LightBridgeConnectService) EncryptToken(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts a system token
func (s *LightBridgeConnectService) DecryptToken(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// VerifyNewAPIToken verifies a New API system token and returns user info.
// userID is sent as the New-Api-User header, which New API requires when
// authenticating web endpoints (e.g. /api/user/self) with a system access token.
func (s *LightBridgeConnectService) VerifyNewAPIToken(ctx context.Context, instanceURL, token string, userID int) (*NewAPIUserResponse, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Normalize URL: strip trailing slashes and /v1 /v2 path prefixes
	baseURL := strings.TrimRight(instanceURL, "/")
	for _, suffix := range []string{"/v1", "/v2"} {
		if strings.HasSuffix(baseURL, suffix) {
			baseURL = baseURL[:len(baseURL)-len(suffix)]
			break
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/user/self", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	// New API requires the New-Api-User header (the caller's numeric user ID)
	// when accessing web endpoints via a system access token.
	if userID > 0 {
		req.Header.Set("New-Api-User", strconv.Itoa(userID))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to New API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("request failed, status: %d, body: %s", resp.StatusCode, preview)
	}

	var result NewAPIUserResponse
	if err := json.Unmarshal(body, &result); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("failed to parse response (not JSON): %s", preview)
	}

	if !result.Success {
		return nil, fmt.Errorf("new API returned success=false")
	}

	return &result, nil
}

// SyncNewAPIQuota fetches the latest quota from New API
func (s *LightBridgeConnectService) SyncNewAPIQuota(ctx context.Context, config *LightBridgeConnectConfig) (*QuotaInfo, error) {
	// Decrypt token
	token, err := s.DecryptToken(config.SystemToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Fetch user info
	userInfo, err := s.VerifyNewAPIToken(ctx, config.InstanceURL, token, config.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &QuotaInfo{
		Balance:    int64(userInfo.Data.Quota - userInfo.Data.UsedQuota),
		Used:       int64(userInfo.Data.UsedQuota),
		LastSyncAt: &now,
		Currency:   "CNY", // New API uses CNY (cents/fen)
	}, nil
}

// CheckQuotaAlert checks if quota alert should be triggered
func (s *LightBridgeConnectService) CheckQuotaAlert(config *LightBridgeConnectConfig, oldBalance, newBalance int64) *AlertInfo {
	if config.Alert == nil || !config.Alert.Enabled {
		return nil
	}

	threshold := config.Alert.Threshold

	// Quota crossed threshold from sufficient to insufficient
	if oldBalance >= threshold && newBalance < threshold {
		return &AlertInfo{
			Type:     "quota_low",
			Severity: "warning",
			Message:  fmt.Sprintf("Quota low: %.2f CNY remaining (threshold: %.2f CNY)", float64(newBalance)/100, float64(threshold)/100),
		}
	}

	// Quota exhausted
	if newBalance <= 0 {
		return &AlertInfo{
			Type:     "quota_exhausted",
			Severity: "critical",
			Message:  "Quota exhausted: account balance is zero or negative",
		}
	}

	return nil
}

// AlertInfo represents an alert to be sent
type AlertInfo struct {
	Type     string
	Severity string
	Message  string
	Metadata map[string]interface{}
}

// SendAlert sends alert through configured channels
func (s *LightBridgeConnectService) SendAlert(ctx context.Context, accountID int64, config *LightBridgeConnectConfig, alert *AlertInfo) error {
	if config.Alert == nil {
		return nil
	}

	for _, channel := range config.Alert.Channels {
		switch channel {
		case "email":
			// TODO: Implement email notification
		case "webhook":
			if config.WebhookURL != "" {
				if err := s.sendWebhook(ctx, config.WebhookURL, accountID, alert); err != nil {
					// Log error but continue with other channels
					continue
				}
			}
		case "dashboard":
			// TODO: Create dashboard notification
		}
	}

	return nil
}

// sendWebhook sends webhook notification
func (s *LightBridgeConnectService) sendWebhook(ctx context.Context, webhookURL string, accountID int64, alert *AlertInfo) error {
	payload := map[string]interface{}{
		"account_id": accountID,
		"type":       alert.Type,
		"severity":   alert.Severity,
		"message":    alert.Message,
		"metadata":   alert.Metadata,
		"timestamp":  time.Now().Unix(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewReader(body))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
