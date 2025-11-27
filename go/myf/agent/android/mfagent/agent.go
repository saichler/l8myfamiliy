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

var (
	deviceID      = ""
	deviceName    = ""
	website       = ""
	user          = ""
	pass          = ""
	bearerToken   = ""
	configDir     = ""
	skipTLSVerify = false
	initialized   = false
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

// Authenticate performs authentication against the server.
// Returns an error if authentication fails.
func Authenticate() error {
	if website == "" {
		return fmt.Errorf("website not configured")
	}
	if user == "" || pass == "" {
		return fmt.Errorf("credentials not configured")
	}

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

	token := strings.TrimSpace(string(body))
	if token == "" {
		return fmt.Errorf("authentication failed: empty response")
	}

	bearerToken = token
	initialized = true
	return nil
}

// RegisterDevice registers the device with the server.
// Must be called after Authenticate.
func RegisterDevice() error {
	if bearerToken == "" {
		return fmt.Errorf("not authenticated")
	}

	deviceEndpoint := strings.TrimSuffix(website, "/") + "/probler/53/Family"

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
func Initialize() error {
	if err := LoadConfig(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if NeedsConfiguration() {
		return fmt.Errorf("configuration required: website or credentials not set")
	}

	if err := Authenticate(); err != nil {
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

	locationEndpoint := strings.TrimSuffix(website, "/") + "/probler/53/Location"

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
