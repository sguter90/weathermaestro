package netatmo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// Puller implements the Netatmo weather data puller
type Puller struct {
	client    *Client
	deviceID  string
	dbManager *database.DatabaseManager
	stationID uuid.UUID
}

// NewPuller creates a new Netatmo puller with database connection
func NewPuller(dbManger *database.DatabaseManager) *Puller {
	return &Puller{
		dbManager: dbManger,
	}
	// replace DB with dbManager manager
}

func (p *Puller) GetProviderType() string {
	return "netatmo"
}

func (p *Puller) ValidateConfig(config map[string]string) error {
	requiredFields := []string{"client_id", "client_secret", "redirect_uri", "device_id", "access_token", "refresh_token", "token_expiry"}
	for _, field := range requiredFields {
		if config[field] == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func (p *Puller) Pull(ctx context.Context, config map[string]string) (map[string]models.SensorReading, *models.StationData, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, nil, err
	}

	p.deviceID = config["device_id"]

	// Load the station ID from database
	if err := p.loadStationID(ctx, p.deviceID); err != nil {
		return nil, nil, fmt.Errorf("failed to load station ID: %w", err)
	}

	// Initialize client if needed
	var err error
	if p.client == nil {
		err = p.initClient(config)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Get station data
	netatmoResp, err := p.client.GetStationsData(ctx, p.deviceID)
	if err != nil {
		return nil, nil, err
	}

	if len(netatmoResp.Body.Devices) == 0 {
		return nil, nil, fmt.Errorf("no devices found in Netatmo response")
	}

	device := netatmoResp.Body.Devices[0]

	// Create station data
	stationData := &models.StationData{
		ID:          p.stationID,
		StationType: "netatmo",
	}

	sensors := p.getSensorsFromDevice(device)
	sensors, err = p.dbManager.EnsureSensorsByRemoteId(p.stationID, sensors)
	if err != nil {
		log.Printf("❌ Failed to ensure sensors: %v", err)
		return nil, nil, err
	}

	// Parse indoor data (from main device)
	dateUTC := p.unixToTime(device.DashboardData.TimeUTC)

	sensorReadings := make(map[string]models.SensorReading)

	var remoteId string

	remoteId = device.ID + "-" + models.SensorTypeTemperature
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    device.DashboardData.Temperature,
		DateUTC:  dateUTC,
	}

	remoteId = device.ID + "-" + models.SensorTypeHumidity
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    float64(device.DashboardData.Humidity),
		DateUTC:  dateUTC,
	}

	remoteId = device.ID + "-" + models.SensorTypePressure
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    device.DashboardData.Pressure,
		DateUTC:  dateUTC,
	}

	remoteId = device.ID + "-" + models.SensorTypePressureAbsolute
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    device.DashboardData.AbsolutePressure,
		DateUTC:  dateUTC,
	}

	remoteId = device.ID + "-" + models.SensorTypeCO2
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    float64(device.DashboardData.CO2),
		DateUTC:  dateUTC,
	}

	remoteId = device.ID + "-" + models.SensorTypeNoise
	sensorReadings[remoteId] = models.SensorReading{
		SensorID: sensors[remoteId].ID,
		Value:    float64(device.DashboardData.Noise),
		DateUTC:  dateUTC,
	}

	// Parse outdoor data (from modules)
	for _, module := range device.Modules {
		p.parseModuleToSensorReadings(sensorReadings, sensors, module, dateUTC)
	}

	return sensorReadings, stationData, nil
}

// parseModuleToSensorReadings parses a Netatmo module and adds sensor readings to the map
func (p *Puller) parseModuleToSensorReadings(sensorReadings map[string]models.SensorReading, sensors map[string]models.Sensor, module StationDataModule, dateUTC time.Time) {
	var remoteId string

	switch module.Type {
	case "NAModule1": // Outdoor module
		remoteId = module.ID + "-" + models.SensorTypeTemperatureOutdoor
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    module.DashboardData.Temperature,
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeHumidityOutdoor
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.Humidity),
			DateUTC:  dateUTC,
		}

	case "NAModule2": // Wind gauge
		remoteId = module.ID + "-" + models.SensorTypeWindDirection
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.WindAngle),
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeWindSpeed
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.WindStrength),
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeWindGust
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.GustStrength),
			DateUTC:  dateUTC,
		}

	case "NAModule3": // Rain gauge
		remoteId = module.ID + "-" + models.SensorTypeRainfallRate
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    module.DashboardData.Rain,
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeRainfallHourly
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    module.DashboardData.SumRain1,
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeRainfallDaily
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    module.DashboardData.SumRain24,
			DateUTC:  dateUTC,
		}

	case "NAModule4": // Additional indoor module
		remoteId = module.ID + "-" + models.SensorTypeTemperature
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    module.DashboardData.Temperature,
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeHumidity
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.Humidity),
			DateUTC:  dateUTC,
		}

		remoteId = module.ID + "-" + models.SensorTypeCO2
		sensorReadings[remoteId] = models.SensorReading{
			SensorID: sensors[remoteId].ID,
			Value:    float64(module.DashboardData.CO2),
			DateUTC:  dateUTC,
		}

	default:
		fmt.Printf("Unknown module type: %s\n", module.Type)
	}
}

// touchStation saves the station config in the database
func (p *Puller) touchStation(ctx context.Context, stationID uuid.UUID) error {
	query := `UPDATE stations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := p.dbManager.GetDB().ExecContext(ctx, query, stationID.String())

	return err
}

func (p *Puller) unixToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0).UTC()
}

// loadStationID loads the station ID from the database based on pass_key
func (p *Puller) loadStationID(ctx context.Context, deviceId string) error {
	query := `SELECT id FROM stations WHERE config->>'device_id' = $1 AND station_type = $2`

	err := p.dbManager.GetDB().QueryRowContext(ctx, query, deviceId, "netatmo").Scan(&p.stationID)
	if err != nil {
		return fmt.Errorf("failed to query station ID: %w", err)
	}

	return nil
}

// updateTokensInDatabase updates only the token fields in the station config
func (p *Puller) updateTokensInDatabase(ctx context.Context, accessToken, refreshToken string, expiry time.Time) error {
	query := `UPDATE stations 
              SET config = config || jsonb_build_object(
                    'access_token', $1,
                    'refresh_token', $2,
                    'token_expiry', $3
                  ),
                  updated_at = CURRENT_TIMESTAMP 
              WHERE id = $4`

	result, err := p.dbManager.GetDB().ExecContext(ctx, query,
		accessToken,
		refreshToken,
		expiry.Format(time.RFC3339),
		p.stationID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no station found with ID %s", p.stationID.String())
	}

	return nil
}

// updateConfigForReauthorizationInDatabase clears tokens and updates state for re-authorization
func (p *Puller) updateConfigForReauthorizationInDatabase(ctx context.Context, state string) error {
	query := `UPDATE stations 
              SET config = config || jsonb_build_object(
                    'access_token', null,
                    'refresh_token', null,
                    'token_expiry', null,
                    'state', $1::text
                  ),
                  updated_at = CURRENT_TIMESTAMP 
              WHERE id = $2`

	result, err := p.dbManager.GetDB().ExecContext(ctx, query,
		state,
		p.stationID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update config for reauthorization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no station found with ID %s", p.stationID.String())
	}

	return nil
}

func (p *Puller) getSensorsFromDevice(device StationDataDevice) map[string]models.Sensor {
	sensors := make(map[string]models.Sensor)
	supportedSensors := GetSupportedSensors()

	// Helper function to safely add sensor
	addSensor := func(remoteID, sensorKey string) {
		if sensor, exists := supportedSensors[sensorKey]; exists {
			// Create a copy and set the RemoteID
			sensorCopy := sensor
			sensorCopy.RemoteID = remoteID
			sensors[remoteID] = sensorCopy
		} else {
			log.Printf("⚠️  Warning: Sensor key '%s' not found in supported sensors", sensorKey)
		}
	}

	// Check main device (NAMain) sensors
	addSensor(device.ID+"-"+models.SensorTypeTemperature, "NAMain-"+models.SensorTypeTemperature)
	addSensor(device.ID+"-"+models.SensorTypeHumidity, "NAMain-"+models.SensorTypeHumidity)
	addSensor(device.ID+"-"+models.SensorTypePressure, "NAMain-"+models.SensorTypePressure)
	addSensor(device.ID+"-"+models.SensorTypePressureAbsolute, "NAMain-"+models.SensorTypePressureAbsolute)
	addSensor(device.ID+"-"+models.SensorTypeCO2, "NAMain-"+models.SensorTypeCO2)
	addSensor(device.ID+"-"+models.SensorTypeNoise, "NAMain-"+models.SensorTypeNoise)

	// Check each module
	for _, module := range device.Modules {
		switch module.Type {
		case "NAModule1": // Outdoor module
			addSensor(module.ID+"-"+models.SensorTypeTemperatureOutdoor, "NAModule1-"+models.SensorTypeTemperatureOutdoor)
			addSensor(module.ID+"-"+models.SensorTypeHumidityOutdoor, "NAModule1-"+models.SensorTypeHumidityOutdoor)

		case "NAModule2": // Wind gauge
			addSensor(module.ID+"-"+models.SensorTypeWindDirection, "NAModule2-"+models.SensorTypeWindDirection)
			addSensor(module.ID+"-"+models.SensorTypeWindSpeed, "NAModule2-"+models.SensorTypeWindSpeed)
			addSensor(module.ID+"-"+models.SensorTypeWindGust, "NAModule2-"+models.SensorTypeWindGust)
			addSensor(module.ID+"-"+models.SensorTypeWindGustAngle, "NAModule2-"+models.SensorTypeWindGustAngle)
			addSensor(module.ID+"-"+models.SensorTypeWindSpeedMaxDaily, "NAModule2-"+models.SensorTypeWindSpeedMaxDaily)

		case "NAModule3": // Rain gauge
			addSensor(module.ID+"-"+models.SensorTypeRainfallRate, "NAModule3-"+models.SensorTypeRainfallRate)
			addSensor(module.ID+"-"+models.SensorTypeRainfallDaily, "NAModule3-"+models.SensorTypeRainfallDaily)
			addSensor(module.ID+"-"+models.SensorTypeRainfallHourly, "NAModule3-"+models.SensorTypeRainfallHourly)

		case "NAModule4": // Additional indoor module
			addSensor(module.ID+"-"+models.SensorTypeTemperature, "NAModule4-"+models.SensorTypeTemperature)
			addSensor(module.ID+"-"+models.SensorTypeHumidity, "NAModule4-"+models.SensorTypeHumidity)
			addSensor(module.ID+"-"+models.SensorTypePressure, "NAModule4-"+models.SensorTypePressure)
			addSensor(module.ID+"-"+models.SensorTypeCO2, "NAModule4-"+models.SensorTypeCO2)
			addSensor(module.ID+"-"+models.SensorTypeNoise, "NAModule4-"+models.SensorTypeNoise)

		default:
			log.Printf("⚠️  Warning: Unknown module type: %s", module.Type)
		}
	}

	return sensors
}

func (p *Puller) initClient(config map[string]string) error {
	onTokenInvalid := func(state string) error {
		// Use a new context for database operations to avoid context cancellation from the caller
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.updateConfigForReauthorizationInDatabase(dbCtx, state)
	}

	p.client = NewClient(
		config["client_id"],
		config["client_secret"],
		config["redirect_uri"],
	)

	tokenExpiryString, ok := config["token_expiry"]
	var tokenExpiry time.Time
	var err error
	if !ok || tokenExpiryString == "" {
		err = fmt.Errorf("token expiry not available")
	}
	if err == nil {
		tokenExpiry, err = time.Parse(time.RFC3339, tokenExpiryString)
	}
	if err != nil {
		authUrl, state := p.client.GetAuthorizationURL()
		err := onTokenInvalid(state)
		if err != nil {
			return err
		}
		return fmt.Errorf("token expiry invalid - you need to re-authorize by visiting the authorization URL: %s", authUrl)
	}

	p.client.SetAccessToken(config["access_token"])
	p.client.SetRefreshToken(config["refresh_token"])
	p.client.SetTokenExpiry(tokenExpiry)
	p.client.SetTokenRefreshCallback(func(accessToken, refreshToken string, expiry time.Time) error {
		// Use a new context for database operations to avoid context cancellation from the caller
		dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.updateTokensInDatabase(dbCtx, accessToken, refreshToken, expiry)
	})
	p.client.SetTokenInvalidCallback(onTokenInvalid)

	return nil
}
