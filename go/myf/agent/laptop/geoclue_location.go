package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
)

// getLocationFromGeoClue gets location using GeoClue2 D-Bus service
// GeoClue is the standard location service on Linux desktops (GNOME, KDE, etc.)
// It can use WiFi positioning, GPS (if available), and IP-based geolocation
func getLocationFromGeoClue() (*l8myfamily.Location, error) {
	// First check if GeoClue service is available
	if !isGeoClueAvailable() {
		return nil, fmt.Errorf("GeoClue2 service not available")
	}

	// Try using the 'where-am-i' command if available (part of geoclue-2.0-demos)
	location, err := getLocationFromWhereAmI()
	if err == nil {
		return location, nil
	}

	// Fall back to direct D-Bus call via gdbus
	return getLocationFromGDBus()
}

// isGeoClueAvailable checks if the GeoClue2 D-Bus service is running
func isGeoClueAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "introspect", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", "/org/freedesktop/GeoClue2/Manager")
	err := cmd.Run()
	return err == nil
}

// getLocationFromWhereAmI uses the 'where-am-i' utility from geoclue-2.0-demos
func getLocationFromWhereAmI() (*l8myfamily.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "where-am-i", "-t", "10")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("where-am-i failed: %w", err)
	}

	return parseWhereAmIOutput(string(output))
}

// parseWhereAmIOutput parses the output from where-am-i command
func parseWhereAmIOutput(output string) (*l8myfamily.Location, error) {
	latRegex := regexp.MustCompile(`Latitude:\s+([+-]?\d+\.?\d*)`)
	lonRegex := regexp.MustCompile(`Longitude:\s+([+-]?\d+\.?\d*)`)

	latMatch := latRegex.FindStringSubmatch(output)
	lonMatch := lonRegex.FindStringSubmatch(output)

	if latMatch == nil || lonMatch == nil {
		return nil, fmt.Errorf("could not parse location from where-am-i output")
	}

	lat, err := strconv.ParseFloat(latMatch[1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latitude: %w", err)
	}

	lon, err := strconv.ParseFloat(lonMatch[1], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse longitude: %w", err)
	}

	return &l8myfamily.Location{
		Latitude:  float32(lat),
		Longitude: float32(lon),
	}, nil
}

// getLocationFromGDBus gets location directly via D-Bus using gdbus command
func getLocationFromGDBus() (*l8myfamily.Location, error) {
	// Step 1: Get a client from the GeoClue Manager
	clientPath, err := getGeoClueClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get GeoClue client: %w", err)
	}

	// Step 2: Set the desktop ID (required for authorization)
	// Try multiple desktop IDs - our app first, then common system apps as fallback
	desktopIDs := []string{
		"l8myfamily-agent",
		"org.gnome.Shell",
		"gnome-shell",
		"firefox",
		"org.mozilla.firefox",
		"chromium",
		"google-chrome",
	}
	var desktopErr error
	for _, desktopID := range desktopIDs {
		desktopErr = setGeoClueDesktopID(clientPath, desktopID)
		if desktopErr == nil {
			break
		}
	}
	if desktopErr != nil {
		return nil, fmt.Errorf("failed to set desktop ID (tried %d options): %w", len(desktopIDs), desktopErr)
	}

	// Step 3: Set the requested accuracy level (EXACT = 8)
	err = setGeoClueAccuracyLevel(clientPath, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to set accuracy level: %w", err)
	}

	// Step 4: Start the client
	err = startGeoClueClient(clientPath)
	if err != nil {
		return nil, fmt.Errorf("failed to start GeoClue client: %w", err)
	}
	defer stopGeoClueClient(clientPath)

	// Step 5: Wait for location and get it
	return waitForGeoClueLocation(clientPath)
}

// getGeoClueClient requests a new client from GeoClue Manager
func getGeoClueClient() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", "/org/freedesktop/GeoClue2/Manager",
		"--method", "org.freedesktop.GeoClue2.Manager.GetClient")

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format: (objectpath '/org/freedesktop/GeoClue2/Client/1',)
	re := regexp.MustCompile(`'(/org/freedesktop/GeoClue2/Client/\d+)'`)
	matches := re.FindStringSubmatch(string(output))
	if matches == nil {
		return "", fmt.Errorf("could not parse client path from: %s", output)
	}

	return matches[1], nil
}

// setGeoClueDesktopID sets the DesktopId property for authorization
func setGeoClueDesktopID(clientPath, desktopID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", clientPath,
		"--method", "org.freedesktop.DBus.Properties.Set",
		"org.freedesktop.GeoClue2.Client", "DesktopId",
		fmt.Sprintf("<'%s'>", desktopID))

	return cmd.Run()
}

// setGeoClueAccuracyLevel sets the requested accuracy level
// Levels: NONE=0, COUNTRY=1, CITY=4, NEIGHBORHOOD=5, STREET=6, EXACT=8
func setGeoClueAccuracyLevel(clientPath string, level int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", clientPath,
		"--method", "org.freedesktop.DBus.Properties.Set",
		"org.freedesktop.GeoClue2.Client", "RequestedAccuracyLevel",
		fmt.Sprintf("<uint32 %d>", level))

	return cmd.Run()
}

// startGeoClueClient starts the location client
func startGeoClueClient(clientPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", clientPath,
		"--method", "org.freedesktop.GeoClue2.Client.Start")

	return cmd.Run()
}

// stopGeoClueClient stops the location client
func stopGeoClueClient(clientPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", clientPath,
		"--method", "org.freedesktop.GeoClue2.Client.Stop")

	cmd.Run()
}

// waitForGeoClueLocation waits for location to be available and retrieves it
func waitForGeoClueLocation(clientPath string) (*l8myfamily.Location, error) {
	// Poll for location up to 15 seconds
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)

		locationPath, err := getGeoClueLocationPath(clientPath)
		if err != nil || locationPath == "" || locationPath == "/" {
			continue
		}

		return getGeoClueLocationData(locationPath)
	}

	return nil, fmt.Errorf("timeout waiting for GeoClue location")
}

// getGeoClueLocationPath gets the current Location object path from the client
func getGeoClueLocationPath(clientPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", clientPath,
		"--method", "org.freedesktop.DBus.Properties.Get",
		"org.freedesktop.GeoClue2.Client", "Location")

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format: (<objectpath '/org/freedesktop/GeoClue2/Client/1/Location'>,)
	re := regexp.MustCompile(`'(/org/freedesktop/GeoClue2/[^']+)'`)
	matches := re.FindStringSubmatch(string(output))
	if matches == nil {
		return "", fmt.Errorf("could not parse location path")
	}

	return matches[1], nil
}

// getGeoClueLocationData retrieves latitude and longitude from a Location object
func getGeoClueLocationData(locationPath string) (*l8myfamily.Location, error) {
	lat, err := getGeoClueProperty(locationPath, "Latitude")
	if err != nil {
		return nil, fmt.Errorf("failed to get latitude: %w", err)
	}

	lon, err := getGeoClueProperty(locationPath, "Longitude")
	if err != nil {
		return nil, fmt.Errorf("failed to get longitude: %w", err)
	}

	return &l8myfamily.Location{
		Latitude:  float32(lat),
		Longitude: float32(lon),
	}, nil
}

// getGeoClueProperty gets a double property from a GeoClue Location object
func getGeoClueProperty(objectPath, property string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gdbus", "call", "--system",
		"--dest", "org.freedesktop.GeoClue2",
		"--object-path", objectPath,
		"--method", "org.freedesktop.DBus.Properties.Get",
		"org.freedesktop.GeoClue2.Location", property)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Output format: (<double 37.7749>,) or (<37.7749>,)
	outputStr := strings.TrimSpace(string(output))
	outputStr = strings.TrimPrefix(outputStr, "(<")
	outputStr = strings.TrimPrefix(outputStr, "double ")
	outputStr = strings.TrimSuffix(outputStr, ">,)")
	outputStr = strings.TrimSuffix(outputStr, ",)")

	return strconv.ParseFloat(outputStr, 64)
}

// GeoClueLocationJSON is used for parsing JSON output from some geoclue tools
type GeoClueLocationJSON struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy"`
}

// parseGeoClueJSON parses JSON location output
func parseGeoClueJSON(data string) (*l8myfamily.Location, error) {
	var loc GeoClueLocationJSON
	if err := json.Unmarshal([]byte(data), &loc); err != nil {
		return nil, err
	}

	if loc.Latitude == 0 && loc.Longitude == 0 {
		return nil, fmt.Errorf("invalid location data")
	}

	return &l8myfamily.Location{
		Latitude:  float32(loc.Latitude),
		Longitude: float32(loc.Longitude),
	}, nil
}
