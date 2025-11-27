package main

import (
	"bufio"
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
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"golang.org/x/term"
)

var (
	deviceID      = ""
	deviceName    = ""
	website       = ""
	user          = ""
	pass          = ""
	bearerToken   = ""
	configFile    = ""
	skipTLSVerify = false
)

type Config struct {
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name,omitempty"`
	Website       string `json:"website,omitempty"`
	EncryptedUser string `json:"encrypted_user,omitempty"`
	EncryptedPass string `json:"encrypted_pass,omitempty"`
	SkipTLSVerify *bool  `json:"skip_tls_verify,omitempty"`
}

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	configFile = filepath.Join(configDir, "l8myfamily", "laptop-agent.json")
}

func getEncryptionKey() []byte {
	h := sha256.New()
	h.Write([]byte(deviceID))
	h.Write([]byte("l8myfamily-laptop-agent"))
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

func promptForInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptForPassword(prompt string) string {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		log.Printf("Error reading password: %v", err)
		return ""
	}
	return string(password)
}

func promptForYesNo(prompt string) bool {
	input := promptForInput(prompt + " (y/n): ")
	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
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

func loadOrCreateConfig() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return createNewConfig()
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.DeviceID == "" {
		deviceID = uuid.New().String()
		// Prompt for device name when creating new device ID
		deviceName = promptForInput("Enter device name (e.g., My Laptop): ")
	} else {
		deviceID = cfg.DeviceID
		deviceName = cfg.DeviceName
	}

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

	needsSave := false
	if website == "" {
		website = promptForInput("Enter website URL (e.g., https://example.com): ")
		needsSave = true
	}
	if cfg.SkipTLSVerify == nil {
		validateCert := promptForYesNo("Validate server certificate?")
		skipTLSVerify = !validateCert
		needsSave = true
	}
	if user == "" || pass == "" {
		user = promptForInput("Enter username: ")
		pass = promptForPassword("Enter password: ")
		needsSave = true
	}

	if needsSave {
		if err := saveConfig(); err != nil {
			return err
		}
	}

	return nil
}

func createNewConfig() error {
	deviceID = uuid.New().String()
	log.Printf("Generated new device ID: %s", deviceID)

	deviceName = promptForInput("Enter device name (e.g., My Laptop): ")
	website = promptForInput("Enter website URL (e.g., https://example.com): ")
	validateCert := promptForYesNo("Validate server certificate?")
	skipTLSVerify = !validateCert
	user = promptForInput("Enter username: ")
	pass = promptForPassword("Enter password: ")

	return saveConfig()
}

func saveConfig() error {
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

	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	log.Printf("Config saved to: %s", configFile)
	return nil
}

func authenticate() error {
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
	log.Printf("Authentication successful")
	return nil
}

func registerDevice() error {
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

	log.Printf("Device registered: %s (%s)", deviceName, deviceID)
	return nil
}

func main() {
	if err := loadOrCreateConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := authenticate(); err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	if err := registerDevice(); err != nil {
		log.Fatalf("Failed to register device: %v", err)
	}

	locationEndpoint := strings.TrimSuffix(website, "/") + "/probler/53/Location"
	log.Printf("Starting location agent for device: %s", deviceID)
	log.Printf("Posting to endpoint: %s", locationEndpoint)
	log.Printf("Using free location services (GeoClue -> IP geolocation fallback)")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	collectAndPost()

	for {
		select {
		case <-ticker.C:
			collectAndPost()
		case <-sigChan:
			log.Println("Shutting down location agent...")
			return
		}
	}
}

func collectAndPost() {
	location, err := getLocation()
	if err != nil {
		log.Printf("Error getting location: %v", err)
		return
	}

	location.DeviceId = deviceID

	err = postLocation(location)
	if err != nil {
		log.Printf("Error posting location: %v", err)
		return
	}

	log.Printf("Posted location: lat=%.6f, lon=%.6f", location.Latitude, location.Longitude)
}

func getLocation() (*l8myfamily.Location, error) {
	// Try GeoClue first (Linux system location service - most accurate when available)
	location, err := getLocationFromGeoClue()
	if err == nil {
		log.Printf("Location obtained via GeoClue")
		return location, nil
	}
	log.Printf("GeoClue failed: %v, falling back to IP-based", err)

	// Fall back to IP-based geolocation (free, but city-level accuracy only)
	location, err = getLocationFromGeoIP()
	if err != nil {
		return nil, fmt.Errorf("all location methods failed: %w", err)
	}
	log.Printf("Location obtained via IP geolocation")
	return location, nil
}

type geoIPResponse struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func getLocationFromGeoIP() (*l8myfamily.Location, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("http://ip-api.com/json/")
	if err != nil {
		return nil, fmt.Errorf("geoip request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var geoResp geoIPResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &l8myfamily.Location{
		Latitude:  float32(geoResp.Lat),
		Longitude: float32(geoResp.Lon),
	}, nil
}

func postLocation(location *l8myfamily.Location) error {
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
