/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package mfagent provides location posting functionality for the Android agent.
// This package is designed to be compiled with gomobile bind for Android.
package mfagent

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DefaultEndpoint = "https://www.probler.dev:9092"

var (
	deviceID        = ""
	deviceName      = ""
	website         = DefaultEndpoint
	user            = ""
	pass            = ""
	bearerToken     = ""
	pendingTfaToken = ""
	configDir       = ""
	skipTLSVerify   = false
	initialized     = false
	tfaRequired     = false
)

// Config holds the persistent configuration
type Config struct {
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name,omitempty"`
	Website       string `json:"website,omitempty"`
	EncryptedUser string `json:"encrypted_user,omitempty"`
	EncryptedPass string `json:"encrypted_pass,omitempty"`
	SkipTLSVerify *bool  `json:"skip_tls_verify,omitempty"`
}

// Location represents a GPS location to post
type Location struct {
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AuthResponse represents the response from the /auth endpoint
type AuthResponse struct {
	Token    string `json:"token"`
	NeedTfa  bool   `json:"needTfa"`
	SetupTfa bool   `json:"setupTfa"`
}

// TfaVerifyRequest represents the request body for TFA verification
type TfaVerifyRequest struct {
	UserID string `json:"userId"`
	Code   string `json:"code"`
	Bearer string `json:"bearer"`
}

// TfaVerifyResponse represents the response from the /tfaVerify endpoint
type TfaVerifyResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// SetConfigDir sets the directory where config file will be stored.
// This should be called first, before any other functions.
func SetConfigDir(dir string) {
	configDir = dir
}

// SetWebsite sets the server website URL
func SetWebsite(url string) {
	website = url
}

// SetCredentials sets the username and password for authentication
func SetCredentials(username, password string) {
	user = username
	pass = password
}

// SetSkipTLSVerify sets whether to skip TLS certificate verification
func SetSkipTLSVerify(skip bool) {
	skipTLSVerify = skip
}

// GetSkipTLSVerify returns whether TLS certificate verification is skipped
func GetSkipTLSVerify() bool {
	return skipTLSVerify
}

// GetDeviceID returns the current device ID
func GetDeviceID() string {
	return deviceID
}

// GetUser returns the current username (for display purposes)
func GetUser() string {
	return user
}

// HasCredentials returns true if both username and password are set
func HasCredentials() bool {
	return user != "" && pass != ""
}

// IsTfaRequired returns true if TFA verification is pending
func IsTfaRequired() bool {
	return tfaRequired
}

// ClearTfaState clears the TFA pending state
func ClearTfaState() {
	tfaRequired = false
	pendingTfaToken = ""
}

// IsTfaError returns true if the error indicates TFA is required
func IsTfaError(err error) bool {
	return err != nil && err.Error() == "TFA_REQUIRED"
}

// SetDeviceID sets the device ID
func SetDeviceID(id string) {
	deviceID = id
}

// GetDeviceName returns the current device name
func GetDeviceName() string {
	return deviceName
}

// SetDeviceName sets the device name
func SetDeviceName(name string) {
	deviceName = name
}

// GetWebsite returns the current website URL
func GetWebsite() string {
	return website
}

// GetEndpoint returns the current website URL (alias for GetWebsite)
func GetEndpoint() string {
	return website
}

// SetEndpoint sets the server endpoint URL (alias for SetWebsite)
func SetEndpoint(url string) {
	website = url
}

// IsInitialized returns whether the agent has been initialized
func IsInitialized() bool {
	return initialized
}

// NeedsConfiguration returns true if website or credentials are not set
func NeedsConfiguration() bool {
	return website == "" || user == "" || pass == ""
}

func getConfigPath() string {
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "mfagent-config.json")
}

func getEncryptionKey() []byte {
	h := sha256.New()
	h.Write([]byte(deviceID))
	h.Write([]byte("l8myfamily-android-agent"))
	return h.Sum(nil)
}

func encrypt(plaintext string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encoded string) (string, error) {
	key := getEncryptionKey()
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func getHTTPClient() *http.Client {
	if skipTLSVerify {
		return &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// LoadConfig loads the configuration from the config file.
// Returns an error if the config file doesn't exist or can't be read.
func LoadConfig() error {
	configPath := getConfigPath()
	if configPath == "" {
		return fmt.Errorf("config directory not set")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Generate device ID for new installs
			deviceID = uuid.New().String()
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.DeviceID == "" {
		deviceID = uuid.New().String()
	} else {
		deviceID = cfg.DeviceID
	}

	deviceName = cfg.DeviceName
	website = cfg.Website
	if cfg.SkipTLSVerify != nil {
		skipTLSVerify = *cfg.SkipTLSVerify
	}

	if cfg.EncryptedUser != "" {
		decrypted, err := decrypt(cfg.EncryptedUser)
		if err == nil {
			user = decrypted
		}
	}
	if cfg.EncryptedPass != "" {
		decrypted, err := decrypt(cfg.EncryptedPass)
		if err == nil {
			pass = decrypted
		}
	}

	return nil
}

// SaveConfig saves the current configuration to the config file.
// Credentials are encrypted before saving.
func SaveConfig() error {
	configPath := getConfigPath()
	if configPath == "" {
		return fmt.Errorf("config directory not set")
	}

	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	encryptedUser, err := encrypt(user)
	if err != nil {
		return fmt.Errorf("failed to encrypt user: %w", err)
	}
	encryptedPass, err := encrypt(pass)
	if err != nil {
		return fmt.Errorf("failed to encrypt pass: %w", err)
	}

	cfg := Config{
		DeviceID:      deviceID,
		DeviceName:    deviceName,
		Website:       website,
		EncryptedUser: encryptedUser,
		EncryptedPass: encryptedPass,
		SkipTLSVerify: &skipTLSVerify,
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ErrTfaRequired is returned when TFA verification is needed
var ErrTfaRequired = fmt.Errorf("TFA_REQUIRED")

// Authenticate performs authentication against the server.
// Returns ErrTfaRequired if TFA verification is needed (call VerifyTfa next).
// Returns an error if authentication fails.
func Authenticate() error {
	if website == "" {
		return fmt.Errorf("website not configured")
	}
	if user == "" || pass == "" {
		return fmt.Errorf("credentials not configured")
	}

	// Clear any previous TFA state
	tfaRequired = false
	pendingTfaToken = ""

	authURL := strings.TrimSuffix(website, "/") + "/auth"

	authReq := map[string]string{
		"user": user,
		"pass": pass,
	}
	data, err := json.Marshal(authReq)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	client := getHTTPClient()
	resp, err := client.Post(authURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: %s", strings.TrimSpace(string(body)))
	}

	// Try to parse as JSON first (for TFA detection)
	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err == nil {
		// Check if TFA is required
		if authResp.NeedTfa {
			tfaRequired = true
			pendingTfaToken = authResp.Token
			return ErrTfaRequired
		}

		// Check if TFA setup is required (first time login with TFA)
		if authResp.SetupTfa {
			// For mobile, we don't support TFA setup - user must set up TFA via web
			return fmt.Errorf("TFA setup required - please complete TFA setup via web browser first")
		}

		// Normal successful auth with token in JSON
		if authResp.Token != "" {
			bearerToken = authResp.Token
			initialized = true
			return nil
		}
	}

	// Fallback: treat response as plain token string (legacy support)
	token := strings.TrimSpace(string(body))
	if token == "" {
		return fmt.Errorf("authentication failed: empty response")
	}

	bearerToken = token
	initialized = true
	return nil
}

// VerifyTfa verifies the TFA code after Authenticate returns ErrTfaRequired.
// The code should be a 6-digit string from the authenticator app.
// Returns nil on success, error otherwise.
func VerifyTfa(code string) error {
	if !tfaRequired || pendingTfaToken == "" {
		return fmt.Errorf("no TFA verification pending")
	}

	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return fmt.Errorf("invalid TFA code: must be 6 digits")
	}

	tfaURL := strings.TrimSuffix(website, "/") + "/tfaVerify"

	tfaReq := TfaVerifyRequest{
		UserID: user,
		Code:   code,
		Bearer: pendingTfaToken,
	}
	data, err := json.Marshal(tfaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal TFA request: %w", err)
	}

	client := getHTTPClient()
	resp, err := client.Post(tfaURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("TFA verification request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read TFA response: %w", err)
	}

	var tfaResp TfaVerifyResponse
	if err := json.Unmarshal(body, &tfaResp); err != nil {
		return fmt.Errorf("failed to parse TFA response: %w", err)
	}

	if !tfaResp.Ok {
		errMsg := tfaResp.Error
		if errMsg == "" {
			errMsg = "invalid verification code"
		}
		return fmt.Errorf("TFA verification failed: %s", errMsg)
	}

	// TFA verification successful - use the pending token as bearer token
	bearerToken = pendingTfaToken
	initialized = true
	tfaRequired = false
	pendingTfaToken = ""

	return nil
}

// RegisterDevice registers the device with the server.
// Must be called after Authenticate.
func RegisterDevice() error {
	if bearerToken == "" {
		return fmt.Errorf("not authenticated")
	}

	deviceEndpoint := strings.TrimSuffix(website, "/") + "/my-family/53/Family"

	deviceReq := map[string]string{
		"id":       deviceID,
		"familyId": user,
		"name":     deviceName,
	}
	data, err := json.Marshal(deviceReq)
	if err != nil {
		return fmt.Errorf("failed to marshal device request: %w", err)
	}

	req, err := http.NewRequest("POST", deviceEndpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("device registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Initialize loads config and authenticates with the server.
// This is a convenience function that combines LoadConfig and Authenticate.
// Returns ErrTfaRequired if TFA verification is needed (call VerifyTfa next).
func Initialize() error {
	if err := LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if NeedsConfiguration() {
		return fmt.Errorf("configuration required: website or credentials not set")
	}

	if err := Authenticate(); err != nil {
		// Pass through ErrTfaRequired so caller can handle TFA
		if err == ErrTfaRequired {
			return ErrTfaRequired
		}
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	return nil
}

// PostLocation posts a GPS location to the server.
// The agent must be initialized before calling this function.
func PostLocation(latitude, longitude float64) error {
	if !initialized {
		return fmt.Errorf("agent not initialized")
	}

	location := &Location{
		DeviceID:  deviceID,
		Latitude:  latitude,
		Longitude: longitude,
	}

	data, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location: %w", err)
	}

	locationEndpoint := strings.TrimSuffix(website, "/") + "/my-family/53/Location"

	req, err := http.NewRequest("POST", locationEndpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	client := getHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReAuthenticate re-authenticates with the server.
// Use this if the bearer token has expired.
func ReAuthenticate() error {
	return Authenticate()
}
