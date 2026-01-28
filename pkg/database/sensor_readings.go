package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// StoreSensorReading stores a single sensor reading
func (dm *DatabaseManager) StoreSensorReading(sensorID uuid.UUID, value float64, dateUTC time.Time) error {
	query := `
        INSERT INTO sensor_readings (sensor_id, value, date_utc)
        VALUES ($1, $2, $3)
    `

	_, err := dm.ExecWithHealthCheck(context.Background(), query, sensorID, value, dateUTC.Format(time.RFC3339Nano))
	return err
}

// GetSensorReadings retrieves readings for a sensor within a time range
func (dm *DatabaseManager) GetSensorReadings(sensorID uuid.UUID, startTime, endTime time.Time, limit int) ([]models.SensorReading, error) {
	query := `
        SELECT id, sensor_id, value, date_utc
        FROM sensor_readings
        WHERE sensor_id = $1 AND date_utc >= $2 AND date_utc <= $3
        ORDER BY date_utc DESC
        LIMIT $4
    `

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, sensorID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(&reading.ID, &reading.SensorID, &reading.Value, &reading.DateUTC)
		if err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings, rows.Err()
}

// GetReadings retrieves readings with flexible filtering
func (dm *DatabaseManager) GetReadings(params models.ReadingQueryParams) (*models.ReadingsResponse, error) {
	// Build WHERE clause that will be reused for both queries
	whereClause := ""
	args := []interface{}{}
	argCount := 1

	if params.StationID != nil {
		whereClause += fmt.Sprintf(" AND s.station_id = $%d", argCount)
		args = append(args, *params.StationID)
		argCount++
	}

	if len(params.SensorIDs) > 0 {
		placeholders := []string{}
		for _, sensorID := range params.SensorIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, sensorID)
			argCount++
		}
		whereClause += fmt.Sprintf(" AND sr.sensor_id IN (%s)", strings.Join(placeholders, ","))
	}

	if params.SensorType != "" {
		whereClause += fmt.Sprintf(" AND s.sensor_type = $%d", argCount)
		args = append(args, params.SensorType)
		argCount++
	}

	if params.Location != "" {
		whereClause += fmt.Sprintf(" AND s.location = $%d", argCount)
		args = append(args, params.Location)
		argCount++
	}

	if params.StartTime != "" {
		whereClause += fmt.Sprintf(" AND sr.date_utc >= $%d", argCount)
		args = append(args, params.StartTime)
		argCount++
	}

	if params.EndTime != "" {
		whereClause += fmt.Sprintf(" AND sr.date_utc <= $%d", argCount)
		args = append(args, params.EndTime)
		argCount++
	}

	// Get total count first (before adding LIMIT/OFFSET)
	countQuery := `
        SELECT COUNT(*)
        FROM sensor_readings sr
        JOIN sensors s ON sr.sensor_id = s.id
        WHERE 1=1
    ` + whereClause

	var totalCount int
	err := dm.QueryRowWithHealthCheck(context.Background(), countQuery, args...).Scan(&totalCount)
	if err != nil {
		log.Printf("Failed to get total count: %v", err)
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Now build the main query with the same WHERE clause
	query := `
        SELECT 
            sr.id,
            sr.sensor_id,
            sr.value,
            sr.date_utc,
            s.sensor_type,
            s.location,
            s.name,
            s.station_id
        FROM sensor_readings sr
        JOIN sensors s ON sr.sensor_id = s.id
        WHERE 1=1
    ` + whereClause

	// Add ORDER BY
	query += fmt.Sprintf(" ORDER BY sr.date_utc %s", strings.ToUpper(params.Order))

	// Calculate offset from page
	offset := (params.Page - 1) * params.Limit

	// Add LIMIT and OFFSET
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	queryArgs := append(args, params.Limit, offset)

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	readings := []models.SensorReading{}
	for rows.Next() {
		var reading models.SensorReading
		var sensorType, location, name string
		var stationID uuid.UUID

		err := rows.Scan(
			&reading.ID,
			&reading.SensorID,
			&reading.Value,
			&reading.DateUTC,
			&sensorType,
			&location,
			&name,
			&stationID,
		)
		if err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}

		readings = append(readings, reading)
	}

	// Calculate total pages
	totalPages := (totalCount + params.Limit - 1) / params.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ReadingsResponse{
		Data:         readings,
		Total:        totalCount,
		Page:         params.Page,
		Limit:        params.Limit,
		TotalPages:   totalPages,
		HasMore:      params.Page < totalPages,
		IsAggregated: false,
	}, rows.Err()
}

// GetAggregatedReadings retrieves aggregated readings based on time intervals
func (dm *DatabaseManager) GetAggregatedReadings(params models.ReadingQueryParams) (*models.ReadingsResponse, error) {
	// Convert aggregate interval to PostgreSQL interval
	interval := convertAggregateInterval(params.Aggregate)
	if interval == "" {
		return nil, fmt.Errorf("invalid aggregate interval: %s", params.Aggregate)
	}

	// Build aggregation function
	aggFunc := buildAggregateFunction(params.AggregateFunc)

	// Build WHERE clause that will be reused for both count and main query
	whereClause := ""
	var args []interface{}
	argCount := 1

	if params.StationID != nil {
		whereClause += fmt.Sprintf(" AND s.station_id = $%d", argCount)
		args = append(args, *params.StationID)
		argCount++
	}

	if len(params.SensorIDs) > 0 {
		placeholders := []string{}
		for _, sensorID := range params.SensorIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, sensorID)
			argCount++
		}
		whereClause += fmt.Sprintf(" AND sr.sensor_id IN (%s)", strings.Join(placeholders, ","))
	}

	if params.SensorType != "" {
		whereClause += fmt.Sprintf(" AND s.sensor_type = $%d", argCount)
		args = append(args, params.SensorType)
		argCount++
	}

	if params.Location != "" {
		whereClause += fmt.Sprintf(" AND s.location = $%d", argCount)
		args = append(args, params.Location)
		argCount++
	}

	if params.StartTime != "" {
		whereClause += fmt.Sprintf(" AND sr.date_utc >= $%d", argCount)
		args = append(args, params.StartTime)
		argCount++
	}

	if params.EndTime != "" {
		whereClause += fmt.Sprintf(" AND sr.date_utc <= $%d", argCount)
		args = append(args, params.EndTime)
		argCount++
	}

	// Determine GROUP BY clause
	groupByClause := ""
	if params.GroupBy != "" {
		switch params.GroupBy {
		case "sensor":
			groupByClause = "sr.sensor_id"
		case "sensor_type":
			groupByClause = "s.sensor_type"
		case "location":
			groupByClause = "s.location"
		default:
			groupByClause = "sr.sensor_id"
		}
	} else {
		groupByClause = "sr.sensor_id"
	}

	// Get total count of aggregated buckets (before LIMIT/OFFSET)
	countQuery := fmt.Sprintf(`
        SELECT COUNT(*) FROM (
            SELECT 
                date_trunc('%s', sr.date_utc) as time_bucket,
                %s
            FROM sensor_readings sr
            JOIN sensors s ON sr.sensor_id = s.id
            WHERE 1=1
            %s
            GROUP BY time_bucket, %s
        ) as subquery
    `, interval, groupByClause, whereClause, groupByClause)

	var totalCount int
	err := dm.QueryRowWithHealthCheck(context.Background(), countQuery, args...).Scan(&totalCount)
	if err != nil {
		log.Printf("Failed to get total count for aggregated readings: %v", err)
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Build main aggregation query
	query := fmt.Sprintf(`
        SELECT 
            date_trunc('%s', sr.date_utc) as time_bucket,
            sr.sensor_id,
            %s as value,
            COUNT(*) as count,
            MIN(sr.value) as min_value,
            MAX(sr.value) as max_value
        FROM sensor_readings sr
        JOIN sensors s ON sr.sensor_id = s.id
        WHERE 1=1
    `, interval, aggFunc)

	query += whereClause

	// Group by time bucket and sensor info
	query += fmt.Sprintf(" GROUP BY time_bucket, %s", groupByClause)

	// Order by time bucket
	query += fmt.Sprintf(" ORDER BY time_bucket %s", strings.ToUpper(params.Order))

	// Calculate offset from page
	offset := (params.Page - 1) * params.Limit

	// Add LIMIT and OFFSET
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	queryArgs := append(args, params.Limit, offset)

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []models.AggregatedReading
	for rows.Next() {
		var reading models.AggregatedReading

		err := rows.Scan(
			&reading.DateUTC,
			&reading.SensorID,
			&reading.Value,
			&reading.Count,
			&reading.MinValue,
			&reading.MaxValue,
		)
		if err != nil {
			log.Printf("Failed to scan aggregated reading: %v", err)
			continue
		}

		readings = append(readings, reading)
	}

	// Calculate total pages
	totalPages := (totalCount + params.Limit - 1) / params.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ReadingsResponse{
		Data:         readings,
		Total:        totalCount,
		Page:         params.Page,
		Limit:        params.Limit,
		TotalPages:   totalPages,
		HasMore:      params.Page < totalPages,
		IsAggregated: true,
	}, rows.Err()
}

// GetLatestReadings retrieves the latest reading for each sensor matching the filters
func (dm *DatabaseManager) GetLatestReadings(params models.ReadingQueryParams) ([]models.SensorReading, error) {
	query := `
        SELECT DISTINCT ON (sr.sensor_id)
            sr.id,
            sr.sensor_id,
            sr.value,
            sr.date_utc
        FROM sensor_readings sr
        JOIN sensors s ON sr.sensor_id = s.id
        WHERE 1=1
    `

	args := []interface{}{}
	argCount := 1

	// Build WHERE clause dynamically
	if params.StationID != nil {
		query += fmt.Sprintf(" AND s.station_id = $%d", argCount)
		args = append(args, *params.StationID)
		argCount++
	}

	if len(params.SensorIDs) > 0 {
		placeholders := []string{}
		for _, sensorID := range params.SensorIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
			args = append(args, sensorID)
			argCount++
		}
		query += fmt.Sprintf(" AND sr.sensor_id IN (%s)", strings.Join(placeholders, ","))
	}

	if params.SensorType != "" {
		query += fmt.Sprintf(" AND s.sensor_type = $%d", argCount)
		args = append(args, params.SensorType)
		argCount++
	}

	if params.Location != "" {
		query += fmt.Sprintf(" AND s.location = $%d", argCount)
		args = append(args, params.Location)
		argCount++
	}

	// Order by sensor_id first (for DISTINCT ON), then by date_utc DESC to get latest
	query += " ORDER BY sr.sensor_id, sr.date_utc DESC"

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	readings := []models.SensorReading{}
	for rows.Next() {
		var reading models.SensorReading
		err := rows.Scan(
			&reading.ID,
			&reading.SensorID,
			&reading.Value,
			&reading.DateUTC,
		)
		if err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}
		readings = append(readings, reading)
	}

	return readings, rows.Err()
}

// convertAggregateInterval converts user-friendly interval to PostgreSQL interval
func convertAggregateInterval(interval string) string {
	intervals := map[string]string{
		"1m":  "minute",
		"5m":  "5 minutes",
		"15m": "15 minutes",
		"30m": "30 minutes",
		"1h":  "hour",
		"6h":  "6 hours",
		"12h": "12 hours",
		"1d":  "day",
		"1w":  "week",
		"1M":  "month",
	}

	if pgInterval, ok := intervals[interval]; ok {
		return pgInterval
	}
	return ""
}

// buildAggregateFunction builds the SQL aggregate function based on user input
func buildAggregateFunction(funcName string) string {
	switch funcName {
	case "avg":
		return "AVG(sr.value)"
	case "min":
		return "MIN(sr.value)"
	case "max":
		return "MAX(sr.value)"
	case "sum":
		return "SUM(sr.value)"
	case "count":
		return "COUNT(sr.value)"
	case "first":
		return "FIRST_VALUE(sr.value) OVER (PARTITION BY date_trunc ORDER BY sr.date_utc ASC)"
	case "last":
		return "LAST_VALUE(sr.value) OVER (PARTITION BY date_trunc ORDER BY sr.date_utc DESC)"
	default:
		return "AVG(sr.value)" // default to average
	}
}
