package database

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// StoreSensorReading stores a single sensor reading in ClickHouse.
// async_insert is enabled on the connection, so the server buffers and
// flushes small inserts as larger MergeTree parts.
func (dm *DatabaseManager) StoreSensorReading(sensorID uuid.UUID, value float64, dateUTC time.Time) error {
	const query = `INSERT INTO sensor_readings (sensor_id, value, date_utc) VALUES (?, ?, ?)`
	return dm.ch.Conn().AsyncInsert(context.Background(), query, false, sensorID, value, dateUTC.UTC())
}

// GetSensorReadings retrieves readings for a sensor within a time range.
func (dm *DatabaseManager) GetSensorReadings(sensorID uuid.UUID, startTime, endTime time.Time, limit int) ([]models.SensorReading, error) {
	const query = `
		SELECT id, sensor_id, value, date_utc
		FROM sensor_readings
		WHERE sensor_id = ? AND date_utc >= ? AND date_utc <= ?
		ORDER BY date_utc DESC
		LIMIT ?
	`

	ctx := context.Background()
	rows, err := dm.ch.Conn().Query(ctx, query, sensorID, startTime.UTC(), endTime.UTC(), uint64(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var readings []models.SensorReading
	for rows.Next() {
		var r models.SensorReading
		if err := rows.Scan(&r.ID, &r.SensorID, &r.Value, &r.DateUTC); err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}
		readings = append(readings, r)
	}
	return readings, rows.Err()
}

// sensorMetadata is the per-sensor info from Postgres needed to resolve
// readings-side filters (StationID/SensorType/Location) and to re-group
// aggregated results by sensor_type or location.
type sensorMetadata struct {
	SensorID   uuid.UUID
	SensorType string
	Location   string
	StationID  uuid.UUID
}

// resolveSensors returns the set of sensors that match the metadata filters
// in params (StationID, SensorType, Location, SensorIDs). The returned slice
// is empty when no sensors match — callers should treat that as a zero result.
func (dm *DatabaseManager) resolveSensors(params models.ReadingQueryParams) ([]sensorMetadata, error) {
	var conditions []string
	var args []interface{}
	idx := 1

	if params.StationID != nil {
		conditions = append(conditions, fmt.Sprintf("station_id = $%d", idx))
		args = append(args, *params.StationID)
		idx++
	}
	if params.SensorType != "" {
		conditions = append(conditions, fmt.Sprintf("sensor_type = $%d", idx))
		args = append(args, params.SensorType)
		idx++
	}
	if params.Location != "" {
		conditions = append(conditions, fmt.Sprintf("location = $%d", idx))
		args = append(args, params.Location)
		idx++
	}
	if len(params.SensorIDs) > 0 {
		placeholders := make([]string, 0, len(params.SensorIDs))
		for _, id := range params.SensorIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", idx))
			args = append(args, id)
			idx++
		}
		conditions = append(conditions, fmt.Sprintf("id IN (%s)", strings.Join(placeholders, ",")))
	}

	query := "SELECT id, sensor_type, location, station_id FROM sensors"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := dm.QueryWithHealthCheck(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []sensorMetadata
	for rows.Next() {
		var m sensorMetadata
		if err := rows.Scan(&m.SensorID, &m.SensorType, &m.Location, &m.StationID); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetReadings retrieves raw readings with flexible filtering.
func (dm *DatabaseManager) GetReadings(params models.ReadingQueryParams) (*models.ReadingsResponse, error) {
	sensors, err := dm.resolveSensors(params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sensors: %w", err)
	}

	response := &models.ReadingsResponse{
		Data:         []models.SensorReading{},
		Total:        0,
		Page:         params.Page,
		Limit:        params.Limit,
		TotalPages:   1,
		HasMore:      false,
		IsAggregated: false,
	}

	if len(sensors) == 0 {
		return response, nil
	}

	sensorIDs := make([]uuid.UUID, 0, len(sensors))
	for _, s := range sensors {
		sensorIDs = append(sensorIDs, s.SensorID)
	}

	whereClause, args, err := buildReadingsWhere(sensorIDs, params.StartTime, params.EndTime)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	countQuery := "SELECT count() FROM sensor_readings " + whereClause
	var totalCount uint64
	if err := dm.ch.Conn().QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count readings: %w", err)
	}

	order := strings.ToUpper(params.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	offset := uint64((params.Page - 1) * params.Limit)
	limit := uint64(params.Limit)

	dataQuery := fmt.Sprintf(
		`SELECT id, sensor_id, value, date_utc FROM sensor_readings %s ORDER BY date_utc %s LIMIT %d OFFSET %d`,
		whereClause, order, limit, offset,
	)

	rows, err := dm.ch.Conn().Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	readings := []models.SensorReading{}
	for rows.Next() {
		var r models.SensorReading
		if err := rows.Scan(&r.ID, &r.SensorID, &r.Value, &r.DateUTC); err != nil {
			log.Printf("Failed to scan reading: %v", err)
			continue
		}
		readings = append(readings, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := int((totalCount + uint64(params.Limit) - 1) / uint64(params.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	response.Data = readings
	response.Total = int(totalCount)
	response.TotalPages = totalPages
	response.HasMore = params.Page < totalPages
	return response, nil
}

// buildReadingsWhere builds the WHERE clause for readings queries against ClickHouse.
// Time range filters are optional. The sensor list is required (callers guard the empty case).
func buildReadingsWhere(sensorIDs []uuid.UUID, startTime, endTime string) (string, []interface{}, error) {
	args := []interface{}{sensorIDs}
	parts := []string{"sensor_id IN ?"}

	if startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err != nil {
			return "", nil, fmt.Errorf("invalid start_time: %w", err)
		}
		parts = append(parts, "date_utc >= ?")
		args = append(args, t.UTC())
	}
	if endTime != "" {
		t, err := time.Parse(time.RFC3339, endTime)
		if err != nil {
			return "", nil, fmt.Errorf("invalid end_time: %w", err)
		}
		parts = append(parts, "date_utc <= ?")
		args = append(args, t.UTC())
	}

	return "WHERE " + strings.Join(parts, " AND "), args, nil
}

// bucketRow holds the composable per-(sensor, time_bucket) aggregates fetched
// from ClickHouse. Go-side post-aggregation folds these by group key
// (sensor / sensor_type / location) and applies the requested aggregate function.
type bucketRow struct {
	TimeBucket time.Time
	SensorID   uuid.UUID
	Sum        float64
	Count      uint64
	Min        float64
	Max        float64
	FirstValue float64
	FirstDate  time.Time
	LastValue  float64
	LastDate   time.Time
}

// GetAggregatedReadings retrieves aggregated readings grouped by a time bucket
// and (sensor | sensor_type | location).
func (dm *DatabaseManager) GetAggregatedReadings(params models.ReadingQueryParams) (*models.ReadingsResponse, error) {
	bucketExpr, ok := clickhouseBucketExpr(params.Aggregate)
	if !ok {
		return nil, fmt.Errorf("invalid aggregate interval: %s", params.Aggregate)
	}

	sensors, err := dm.resolveSensors(params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve sensors: %w", err)
	}

	response := &models.ReadingsResponse{
		Data:         []models.AggregatedReading{},
		Total:        0,
		Page:         params.Page,
		Limit:        params.Limit,
		TotalPages:   1,
		HasMore:      false,
		IsAggregated: true,
	}

	if len(sensors) == 0 {
		return response, nil
	}

	sensorIDs := make([]uuid.UUID, 0, len(sensors))
	metaBySensor := make(map[uuid.UUID]sensorMetadata, len(sensors))
	for _, s := range sensors {
		sensorIDs = append(sensorIDs, s.SensorID)
		metaBySensor[s.SensorID] = s
	}

	whereClause, args, err := buildReadingsWhere(sensorIDs, params.StartTime, params.EndTime)
	if err != nil {
		return nil, err
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			%s AS time_bucket,
			sensor_id,
			sum(value)               AS sum_value,
			count()                  AS count_value,
			min(value)               AS min_value,
			max(value)               AS max_value,
			argMin(value, date_utc)  AS first_value,
			min(date_utc)            AS first_date,
			argMax(value, date_utc)  AS last_value,
			max(date_utc)            AS last_date
		FROM sensor_readings
		%s
		GROUP BY time_bucket, sensor_id
	`, bucketExpr, whereClause)

	rows, err := dm.ch.Conn().Query(context.Background(), dataQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []bucketRow
	for rows.Next() {
		var b bucketRow
		if err := rows.Scan(
			&b.TimeBucket, &b.SensorID,
			&b.Sum, &b.Count, &b.Min, &b.Max,
			&b.FirstValue, &b.FirstDate,
			&b.LastValue, &b.LastDate,
		); err != nil {
			log.Printf("Failed to scan bucket row: %v", err)
			continue
		}
		buckets = append(buckets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	aggFunc := params.AggregateFunc
	if aggFunc == "" {
		aggFunc = "avg"
	}

	aggregated := foldBuckets(buckets, metaBySensor, params.GroupBy, aggFunc)

	order := strings.ToUpper(params.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}
	sort.SliceStable(aggregated, func(i, j int) bool {
		if order == "ASC" {
			return aggregated[i].DateUTC.Before(aggregated[j].DateUTC)
		}
		return aggregated[i].DateUTC.After(aggregated[j].DateUTC)
	})

	total := len(aggregated)
	totalPages := (total + params.Limit - 1) / params.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	start := (params.Page - 1) * params.Limit
	if start > total {
		start = total
	}
	end := start + params.Limit
	if end > total {
		end = total
	}

	response.Data = aggregated[start:end]
	response.Total = total
	response.TotalPages = totalPages
	response.HasMore = params.Page < totalPages
	return response, nil
}

// foldBuckets re-aggregates per-sensor-per-bucket rows by the requested group key
// and applies the requested aggregate function.
func foldBuckets(buckets []bucketRow, meta map[uuid.UUID]sensorMetadata, groupBy, aggFunc string) []models.AggregatedReading {
	type folded struct {
		dateUTC    time.Time
		sensorID   uuid.UUID
		sensorType string
		location   string
		sum        float64
		count      uint64
		min        float64
		max        float64
		firstVal   float64
		firstDate  time.Time
		lastVal    float64
		lastDate   time.Time
		seen       bool
	}

	type key struct {
		bucket   time.Time
		groupVal string
	}

	groups := make(map[key]*folded)

	for _, b := range buckets {
		m := meta[b.SensorID]
		var groupVal string
		switch groupBy {
		case "sensor_type":
			groupVal = m.SensorType
		case "location":
			groupVal = m.Location
		default:
			groupVal = b.SensorID.String()
		}

		k := key{bucket: b.TimeBucket, groupVal: groupVal}
		f, ok := groups[k]
		if !ok {
			f = &folded{
				dateUTC:    b.TimeBucket,
				sensorID:   b.SensorID,
				sensorType: m.SensorType,
				location:   m.Location,
				min:        b.Min,
				max:        b.Max,
				firstVal:   b.FirstValue,
				firstDate:  b.FirstDate,
				lastVal:    b.LastValue,
				lastDate:   b.LastDate,
				seen:       true,
			}
			groups[k] = f
		}
		f.sum += b.Sum
		f.count += b.Count
		if b.Min < f.min {
			f.min = b.Min
		}
		if b.Max > f.max {
			f.max = b.Max
		}
		if b.FirstDate.Before(f.firstDate) {
			f.firstDate = b.FirstDate
			f.firstVal = b.FirstValue
		}
		if b.LastDate.After(f.lastDate) {
			f.lastDate = b.LastDate
			f.lastVal = b.LastValue
		}
	}

	out := make([]models.AggregatedReading, 0, len(groups))
	for _, f := range groups {
		r := models.AggregatedReading{
			DateUTC:  f.dateUTC,
			Count:    int(f.count),
			MinValue: f.min,
			MaxValue: f.max,
		}
		switch groupBy {
		case "sensor_type":
			r.SensorType = f.sensorType
		case "location":
			r.Location = f.location
		default:
			r.SensorID = f.sensorID
		}
		switch aggFunc {
		case "min":
			r.Value = f.min
		case "max":
			r.Value = f.max
		case "sum":
			r.Value = f.sum
		case "count":
			r.Value = float64(f.count)
		case "first":
			r.Value = f.firstVal
		case "last":
			r.Value = f.lastVal
		case "avg":
			fallthrough
		default:
			if f.count > 0 {
				r.Value = f.sum / float64(f.count)
			}
		}
		out = append(out, r)
	}
	return out
}

// clickhouseBucketExpr returns the ClickHouse expression that buckets date_utc
// at the requested resolution. The second return value is false for unknown intervals.
func clickhouseBucketExpr(interval string) (string, bool) {
	switch interval {
	case "1m":
		return "toStartOfInterval(date_utc, INTERVAL 1 MINUTE)", true
	case "5m":
		return "toStartOfInterval(date_utc, INTERVAL 5 MINUTE)", true
	case "15m":
		return "toStartOfInterval(date_utc, INTERVAL 15 MINUTE)", true
	case "30m":
		return "toStartOfInterval(date_utc, INTERVAL 30 MINUTE)", true
	case "1h":
		return "toStartOfInterval(date_utc, INTERVAL 1 HOUR)", true
	case "6h":
		return "toStartOfInterval(date_utc, INTERVAL 6 HOUR)", true
	case "12h":
		return "toStartOfInterval(date_utc, INTERVAL 12 HOUR)", true
	case "1d":
		return "toStartOfInterval(date_utc, INTERVAL 1 DAY)", true
	case "1w":
		return "toStartOfInterval(date_utc, INTERVAL 1 WEEK)", true
	case "1M":
		return "toStartOfInterval(date_utc, INTERVAL 1 MONTH)", true
	}
	return "", false
}
