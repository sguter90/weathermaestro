package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SensorReading represents a single measurement from a sensor
type SensorReading struct {
	ID       uuid.UUID `json:"id"`
	SensorID uuid.UUID `json:"sensor_id"`
	Value    float64   `json:"value"`
	DateUTC  time.Time `json:"date_utc"`
}

// ReadingQueryParams holds all query parameters for reading queries
type ReadingQueryParams struct {
	StationID     *uuid.UUID
	SensorIDs     []uuid.UUID
	SensorType    string
	Location      string
	StartTime     string
	EndTime       string
	Limit         int
	Page          int
	Order         string
	Aggregate     string
	AggregateFunc string
	Latest        bool
	GroupBy       string
}

// Validate checks if the query parameters are valid
func (p *ReadingQueryParams) Validate() error {
	// Validate aggregate interval
	if p.Aggregate != "" {
		validIntervals := []string{"1m", "5m", "15m", "30m", "1h", "6h", "12h", "1d", "1w", "1M"}
		valid := false
		for _, interval := range validIntervals {
			if p.Aggregate == interval {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid aggregate interval: %s (valid: %s)", p.Aggregate, strings.Join(validIntervals, ", "))
		}
	}

	// Validate aggregate function
	if p.AggregateFunc != "" {
		validFuncs := []string{"avg", "min", "max", "sum", "count", "first", "last"}
		valid := false
		for _, fn := range validFuncs {
			if p.AggregateFunc == fn {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid aggregate function: %s (valid: %s)", p.AggregateFunc, strings.Join(validFuncs, ", "))
		}
	}

	// Validate group_by
	if p.GroupBy != "" {
		validGroupBy := []string{"sensor", "sensor_type", "location"}
		valid := false
		for _, gb := range validGroupBy {
			if p.GroupBy == gb {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid group_by: %s (valid: %s)", p.GroupBy, strings.Join(validGroupBy, ", "))
		}
	}

	// Validate that aggregate and latest are not used together
	if p.Aggregate != "" && p.Latest {
		return fmt.Errorf("cannot use 'aggregate' and 'latest' parameters together")
	}

	// Validate limit
	if p.Limit < 1 || p.Limit > 10000 {
		return fmt.Errorf("limit must be between 1 and 10000")
	}

	// Validate page
	if p.Page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}

	if p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %s (valid: asc, desc)", p.Order)
	}

	return nil
}

type AggregatedReading struct {
	DateUTC    time.Time `json:"dateutc"`
	SensorID   uuid.UUID `json:"sensor_id,omitempty"`
	SensorType string    `json:"sensor_type,omitempty"`
	Location   string    `json:"location,omitempty"`
	Value      float64   `json:"value"`
	Count      int       `json:"count,omitempty"`
	MinValue   float64   `json:"min_value,omitempty"`
	MaxValue   float64   `json:"max_value,omitempty"`
}

type ReadingsResponse struct {
	Data         interface{} `json:"data"`
	Total        int         `json:"total"`
	Page         int         `json:"page"`
	TotalPages   int         `json:"total_pages"`
	Limit        int         `json:"limit"`
	HasMore      bool        `json:"has_more"`
	IsAggregated bool        `json:"is_aggregated"`
}
