package main

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/puller/netatmo"
)

// ServiceConfigCollector handles collection of service-specific configurations
type ServiceConfigCollector struct {
	reader    *bufio.Reader
	dbManager *database.DatabaseManager
}

// NewServiceConfigCollector creates a new ServiceConfigCollector instance
func NewServiceConfigCollector(reader *bufio.Reader, dbManager *database.DatabaseManager) *ServiceConfigCollector {
	return &ServiceConfigCollector{
		reader:    reader,
		dbManager: dbManager,
	}
}

// Collect gathers configuration based on service type
func (scc *ServiceConfigCollector) Collect(serviceName, mode string, stationID uuid.UUID) map[string]interface{} {
	config := make(map[string]interface{})

	switch serviceName {
	case "ecowitt":
		config = scc.collectEcowittConfig()
	case "netatmo":
		config = scc.collectNetatmoConfig(mode, stationID)
	case "ambient":
		config = scc.collectAmbientConfig()
	case "weatherflow":
		config = scc.collectWeatherflowConfig()
	default:
		fmt.Printf("Unknown service: %s\n", serviceName)
	}

	return config
}

// collectEcowittConfig gathers Ecowitt-specific configuration
func (scc *ServiceConfigCollector) collectEcowittConfig() map[string]interface{} {
	config := make(map[string]interface{})

	fmt.Println("\nEcowitt Configuration:")
	fmt.Print("  API Key (optional): ")
	apiKey, _ := scc.reader.ReadString('\n')
	if strings.TrimSpace(apiKey) != "" {
		config["api_key"] = strings.TrimSpace(apiKey)
	}

	return config
}

// collectNetatmoConfig gathers Netatmo-specific configuration
func (scc *ServiceConfigCollector) collectNetatmoConfig(mode string, stationID uuid.UUID) map[string]interface{} {
	config := make(map[string]interface{})

	fmt.Println("\nNetatmo Configuration:")

	fmt.Print("  Client ID: ")
	clientID, _ := scc.reader.ReadString('\n')
	config["client_id"] = strings.TrimSpace(clientID)

	fmt.Print("  Client Secret: ")
	clientSecret, _ := scc.reader.ReadString('\n')
	config["client_secret"] = strings.TrimSpace(clientSecret)

	if mode == "pull" {
		fmt.Print("  Pull Interval (seconds) [300]: ")
		intervalStr, _ := scc.reader.ReadString('\n')
		intervalStr = strings.TrimSpace(intervalStr)
		interval := 300
		if intervalStr != "" {
			fmt.Sscanf(intervalStr, "%d", &interval)
		}
		config["pull_interval"] = interval

		fmt.Print("  Server Public URL [http://localhost:8059]: ")
		publicURL, _ := scc.reader.ReadString('\n')
		publicURL = strings.TrimSpace(publicURL)
		if publicURL == "" {
			publicURL = "http://localhost:8059"
		}

		// Build redirect URI with station ID
		redirectURI := publicURL + "/netatmo/callback/" + stationID.String()
		config["redirect_uri"] = redirectURI

		// Generate authorization URL
		client := netatmo.NewClient(
			strings.TrimSpace(fmt.Sprintf("%v", config["client_id"])),
			strings.TrimSpace(fmt.Sprintf("%v", config["client_secret"])),
			redirectURI,
		)

		authURL, state := client.GetAuthorizationURL("")
		fmt.Println("\n  ⚠️  Please visit this URL to authorize the application:")
		fmt.Printf("  %s\n\n", authURL)

		config["state"] = state

		// Save initial config to database
		if err := scc.dbManager.SetStationConfig(stationID, config); err != nil {
			fmt.Printf("  ⚠️  Error saving Netatmo config: %v\n", err)
			return config
		}

		// Wait for access token to be set via callback
		fmt.Println("\n  ⏳ Waiting for authorization callback...")
		if err := scc.waitForAccessToken(stationID); err != nil {
			fmt.Printf("  ⚠️  Error waiting for token: %v\n", err)
			return config
		}

		// Read access token from database and add to config
		updatedConfig, err := scc.dbManager.GetStationConfig(stationID)
		if err != nil {
			fmt.Printf("  ⚠️  Error getting config: %v\n", err)
			return config
		}

		// Fetch and display available devices
		updatedConfig, err = scc.selectNetatmoDevice(updatedConfig)
		if err != nil {
			fmt.Printf("  ⚠️  Error selecting device: %v\n", err)
			return updatedConfig
		}

		// Save updated config to database
		if err := scc.dbManager.SetStationConfig(stationID, updatedConfig); err != nil {
			fmt.Printf("  ⚠️  Error saving device config: %v\n", err)
		}

		return updatedConfig
	}

	return config
}

// collectAmbientConfig gathers Ambient Weather-specific configuration
func (scc *ServiceConfigCollector) collectAmbientConfig() map[string]interface{} {
	config := make(map[string]interface{})

	fmt.Println("\nAmbient Weather Configuration:")
	fmt.Print("  API Key: ")
	apiKey, _ := scc.reader.ReadString('\n')
	config["api_key"] = strings.TrimSpace(apiKey)

	fmt.Print("  Application Key: ")
	appKey, _ := scc.reader.ReadString('\n')
	config["app_key"] = strings.TrimSpace(appKey)

	return config
}

// collectWeatherflowConfig gathers WeatherFlow-specific configuration
func (scc *ServiceConfigCollector) collectWeatherflowConfig() map[string]interface{} {
	config := make(map[string]interface{})

	fmt.Println("\nWeatherFlow Configuration:")
	fmt.Print("  Token: ")
	token, _ := scc.reader.ReadString('\n')
	config["token"] = strings.TrimSpace(token)

	fmt.Print("  Device ID: ")
	deviceID, _ := scc.reader.ReadString('\n')
	config["device_id"] = strings.TrimSpace(deviceID)

	return config
}

// waitForAccessToken waits for the OAuth2 access token to be set via callback
func (scc *ServiceConfigCollector) waitForAccessToken(stationID uuid.UUID) error {
	fmt.Print("  Press Enter once you've authorized the application: ")
	scc.reader.ReadString('\n')

	// Verify token was set
	config, err := scc.dbManager.GetStationConfig(stationID)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if accessToken, ok := config["access_token"]; ok && accessToken != "" {
		fmt.Println("  ✓ Access token received!")
		return nil
	}

	return fmt.Errorf("access token not set")
}

// selectNetatmoDevice allows user to select a Netatmo device
func (scc *ServiceConfigCollector) selectNetatmoDevice(config map[string]interface{}) (map[string]interface{}, error) {
	accessToken, ok := config["access_token"].(string)
	if !ok || accessToken == "" {
		return config, fmt.Errorf("access token not available")
	}
	refreshToken, ok := config["refresh_token"].(string)
	if !ok || refreshToken == "" {
		return config, fmt.Errorf("refresh token not available")
	}
	tokenExpiryString, ok := config["token_expiry"].(string)
	if !ok || tokenExpiryString == "" {
		return config, fmt.Errorf("token expiry not available")
	}
	tokenExpiry, err := time.Parse(time.RFC3339, tokenExpiryString)
	if err != nil {
		return config, fmt.Errorf("token expiry invalid: " + tokenExpiryString)
	}

	// Create Netatmo client and fetch devices
	clientID := fmt.Sprintf("%v", config["client_id"])
	clientSecret := fmt.Sprintf("%v", config["client_secret"])
	redirectURI := fmt.Sprintf("%v", config["redirect_uri"])

	client := netatmo.NewClient(clientID, clientSecret, redirectURI)
	client.SetAccessToken(accessToken)
	client.SetRefreshToken(refreshToken)
	client.SetTokenExpiry(tokenExpiry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.GetStationsData(ctx, "")
	if err != nil {
		return config, fmt.Errorf("failed to fetch devices from Netatmo: %w", err)
	}

	if len(resp.Body.Devices) == 0 {
		return config, fmt.Errorf("no devices found in your Netatmo account")
	}

	// Display available devices
	fmt.Println("\n  Available Netatmo Devices:")
	for i, device := range resp.Body.Devices {
		fmt.Printf("  [%d] %s (%s)\n", i+1, device.StationName, device.Type)
		fmt.Printf("      ID: %s\n", device.ID)
	}

	// Let user select device
	fmt.Print("\n  Select device number: ")
	selectionStr, _ := scc.reader.ReadString('\n')
	selectionStr = strings.TrimSpace(selectionStr)

	selection := 1
	if selectionStr != "" {
		fmt.Sscanf(selectionStr, "%d", &selection)
	}

	if selection < 1 || selection > len(resp.Body.Devices) {
		return config, fmt.Errorf("invalid selection")
	}

	selectedDevice := resp.Body.Devices[selection-1]

	// Update config with device info
	config["device_id"] = selectedDevice.ID
	config["device_name"] = selectedDevice.StationName
	config["device_type"] = selectedDevice.Type

	fmt.Printf("  ✓ Selected device: %s\n", selectedDevice.StationName)

	return config, nil
}
