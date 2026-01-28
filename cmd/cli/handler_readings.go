package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// getReadingsHandler returns readings with flexible filtering and aggregation
// Query params:
//   - station_id: filter by station UUID
//   - sensor_id: filter by sensor UUID (can be comma-separated list)
//   - sensor_type: filter by sensor type
//   - location: filter by sensor location
//   - start: start time (RFC3339 or Unix timestamp)
//   - end: end time (RFC3339 or Unix timestamp)
//   - limit: max number of results (default: 100, max: 10000)
//   - offset: pagination offset
//   - order: sort order (asc/desc, default: desc)
//   - aggregate: aggregation interval (1m, 5m, 15m, 1h, 6h, 1d, 1w, 1M)
//   - aggregate_func: aggregation function (avg, min, max, sum, count, first, last)
//   - group_by: group results by (sensor, sensor_type, location)
func (rm *RouteManager) getReadingsHandler(w http.ResponseWriter, r *http.Request) {
	params := parseReadingQueryParams(r)

	// Validate parameters
	if err := params.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var result interface{}
	var err error

	// Handle different query modes
	if params.Aggregate != "" {
		result, err = rm.dbManager.GetAggregatedReadings(params)
	} else {
		result, err = rm.dbManager.GetReadings(params)
	}

	if err != nil {
		log.Printf("❌ Failed to query readings: %v", err)
		http.Error(w, "Failed to query readings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// parseReadingQueryParams extracts and parses query parameters from the request
func parseReadingQueryParams(r *http.Request) models.ReadingQueryParams {
	params := models.ReadingQueryParams{
		SensorType:    r.URL.Query().Get("sensor_type"),
		Location:      r.URL.Query().Get("location"),
		StartTime:     r.URL.Query().Get("start"),
		EndTime:       r.URL.Query().Get("end"),
		Limit:         100,    // default
		Page:          1,      // default
		Order:         "desc", // default
		Aggregate:     r.URL.Query().Get("aggregate"),
		AggregateFunc: r.URL.Query().Get("aggregate_func"),
		Latest:        r.URL.Query().Get("latest") == "true",
		GroupBy:       r.URL.Query().Get("group_by"),
	}

	// Parse station_id
	if stationIDStr := r.URL.Query().Get("station_id"); stationIDStr != "" {
		if id, err := uuid.Parse(stationIDStr); err == nil {
			params.StationID = &id
		}
	}

	// Parse sensor_id (can be comma-separated)
	if sensorIDStr := r.URL.Query().Get("sensor_id"); sensorIDStr != "" {
		for _, idStr := range strings.Split(sensorIDStr, ",") {
			if id, err := uuid.Parse(strings.TrimSpace(idStr)); err == nil {
				params.SensorIDs = append(params.SensorIDs, id)
			}
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 10000 {
			params.Limit = l
		}
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			params.Page = o
		}
	}

	// Parse order
	if orderStr := r.URL.Query().Get("order"); orderStr == "asc" || orderStr == "desc" {
		params.Order = orderStr
	}

	// Default aggregate function
	if params.Aggregate != "" && params.AggregateFunc == "" {
		params.AggregateFunc = "avg"
	}

	return params
}
