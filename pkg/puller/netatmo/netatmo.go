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
	dateUTC := p.unixToTime(device.Modules[0].LastMessage)

	// sensor reading mit remote-id statt sensor type

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
		p.parseModuleToSensorReadings(sensorReadings, sensorMap, module, dateUTC)
	}

	return sensorReadings, stationData, nil
}

// parseModuleToSensorReadings parses a Netatmo module and adds sensor readings to the map
func (p *Puller) parseModuleToSensorReadings(sensorReadings map[uuid.UUID]models.SensorReading, sensorMap map[string]uuid.UUID, module StationDataModule, dateUTC time.Time) {
	switch module.Type {
	case "NAModule1": // Outdoor module
		if module.DashboardData.Temperature != 0 {
			if sensorID, ok := sensorMap[models.SensorTypeTemperatureOutdoor]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    module.DashboardData.Temperature,
					DateUTC:  dateUTC,
				}
			}
		}
		if module.DashboardData.Humidity > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeHumidityOutdoor]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    float64(module.DashboardData.Humidity),
					DateUTC:  dateUTC,
				}
			}
		}

	case "NAModule2": // Wind gauge
		if module.DashboardData.WindAngle > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeWindDirection]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    float64(module.DashboardData.WindAngle),
					DateUTC:  dateUTC,
				}
			}
		}
		if module.DashboardData.WindStrength > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeWindSpeed]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    float64(module.DashboardData.WindStrength),
					DateUTC:  dateUTC,
				}
			}
		}
		if module.DashboardData.GustStrength > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeWindGust]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    float64(module.DashboardData.GustStrength),
					DateUTC:  dateUTC,
				}
			}
		}

	case "NAModule3": // Rain gauge
		if module.DashboardData.Rain > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeRainfallRate]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    module.DashboardData.Rain,
					DateUTC:  dateUTC,
				}
			}
		}

	case "NAModule4": // Additional indoor module
		if module.DashboardData.Temperature != 0 {
			if sensorID, ok := sensorMap[models.SensorTypeTemperatureOutdoor]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    module.DashboardData.Temperature,
					DateUTC:  dateUTC,
				}
			}
		}
		if module.DashboardData.Humidity > 0 {
			if sensorID, ok := sensorMap[models.SensorTypeHumidityOutdoor]; ok {
				sensorReadings[sensorID] = models.SensorReading{
					SensorID: sensorID,
					Value:    float64(module.DashboardData.Humidity),
					DateUTC:  dateUTC,
				}
			}
		}
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

	// Check main device (NAMain) sensors
	sensors[device.ID+"-Temperature"] = supportedSensors["NAMain-Temperature"]
	sensors[device.ID+"-Humdity"] = supportedSensors["NAMain-Humidity"]
	sensors[device.ID+"-Pressure"] = supportedSensors["NAMain-Pressure"]
	sensors[device.ID+"-CO2"] = supportedSensors["NAMain-CO2"]
	sensors[device.ID+"-Noise"] = supportedSensors["NAMain-Noise"]

	// Check each module
	for _, module := range device.Modules {
		switch module.Type {
		case "NAModule1": // Outdoor module
			sensors[module.ID+"-Temperature"] = supportedSensors["NAModule1-Temperature"]
			sensors[module.ID+"-Humidity"] = supportedSensors["NAModule1-Humidity"]

		case "NAModule2": // Wind gauge
			sensors[module.ID+"-WindAngle"] = supportedSensors["NAModule2-WindAngle"]
			sensors[module.ID+"-WindStrength"] = supportedSensors["NAModule2-WindStrength"]
			sensors[module.ID+"-GustStrength"] = supportedSensors["NAModule2-GustStrength"]
			sensors[module.ID+"-GustAngle"] = supportedSensors["NAModule2-GustAngle"]
			sensors[module.ID+"-MaxWindStr"] = supportedSensors["NAModule2-MaxWindStr"]

		case "NAModule3": // Rain gauge
			sensors[module.ID+"-Rain"] = supportedSensors["NAModule3-Rain"]
			sensors[module.ID+"-SumRain24"] = supportedSensors["NAModule3-SumRain24"]
			sensors[module.ID+"-SumRain1"] = supportedSensors["NAModule3-SumRain1"]

		case "NAModule4": // Additional indoor module
			sensors[module.ID+"-Temperature"] = supportedSensors["NAModule4-Temperature"]
			sensors[module.ID+"-Humidity"] = supportedSensors["NAModule4-Humidity"]
			sensors[module.ID+"-CO2"] = supportedSensors["NAModule4-CO2"]
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
