package models

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReadingQueryParams_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		params      ReadingQueryParams
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid basic params",
			params: ReadingQueryParams{
				Limit: 100,
				Page:  1,
				Order: "desc",
			},
			expectError: false,
		},
		{
			name: "Valid with aggregation",
			params: ReadingQueryParams{
				Limit:         100,
				Page:          1,
				Order:         "asc",
				Aggregate:     "1h",
				AggregateFunc: "avg",
			},
			expectError: false,
		},
		{
			name: "Invalid limit - too low",
			params: ReadingQueryParams{
				Limit: 0,
				Page:  1,
				Order: "desc",
			},
			expectError: true,
			errorMsg:    "limit must be between 1 and 10000",
		},
		{
			name: "Invalid limit - too high",
			params: ReadingQueryParams{
				Limit: 10001,
				Page:  1,
				Order: "desc",
			},
			expectError: true,
			errorMsg:    "limit must be between 1 and 10000",
		},
		{
			name: "Invalid page - zero",
			params: ReadingQueryParams{
				Limit: 100,
				Page:  0,
				Order: "desc",
			},
			expectError: true,
			errorMsg:    "page must be greater than 0",
		},
		{
			name: "Invalid page - negative",
			params: ReadingQueryParams{
				Limit: 100,
				Page:  -1,
				Order: "desc",
			},
			expectError: true,
			errorMsg:    "page must be greater than 0",
		},
		{
			name: "Invalid order",
			params: ReadingQueryParams{
				Limit: 100,
				Page:  1,
				Order: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid order: invalid (valid: asc, desc)",
		},
		{
			name: "Invalid aggregate interval",
			params: ReadingQueryParams{
				Limit:         100,
				Page:          1,
				Order:         "desc",
				Aggregate:     "invalid",
				AggregateFunc: "avg",
			},
			expectError: true,
			errorMsg:    "invalid aggregate interval",
		},
		{
			name: "Invalid aggregate function",
			params: ReadingQueryParams{
				Limit:         100,
				Page:          1,
				Order:         "desc",
				Aggregate:     "1h",
				AggregateFunc: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid aggregate function",
		},
		{
			name: "Invalid group by",
			params: ReadingQueryParams{
				Limit:   100,
				Page:    1,
				Order:   "desc",
				GroupBy: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid group_by",
		},
		{
			name: "Valid group by - sensor",
			params: ReadingQueryParams{
				Limit:   100,
				Page:    1,
				Order:   "desc",
				GroupBy: "sensor",
			},
			expectError: false,
		},
		{
			name: "Valid group by - sensor_type",
			params: ReadingQueryParams{
				Limit:   100,
				Page:    1,
				Order:   "desc",
				GroupBy: "sensor_type",
			},
			expectError: false,
		},
		{
			name: "Valid group by - location",
			params: ReadingQueryParams{
				Limit:   100,
				Page:    1,
				Order:   "desc",
				GroupBy: "location",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestReadingQueryParams_TimeRangeValidation(t *testing.T) {
	now := time.Now().UTC()
	stationID := uuid.New()

	testCases := []struct {
		name        string
		params      ReadingQueryParams
		expectError bool
	}{
		{
			name: "Valid time range",
			params: ReadingQueryParams{
				StationID: &stationID,
				StartTime: now.Add(-1 * time.Hour).Format(time.RFC3339),
				EndTime:   now.Format(time.RFC3339),
				Limit:     100,
				Page:      1,
				Order:     "desc",
			},
			expectError: false,
		},
		{
			name: "Start time after end time",
			params: ReadingQueryParams{
				StationID: &stationID,
				StartTime: now.Format(time.RFC3339),
				EndTime:   now.Add(-1 * time.Hour).Format(time.RFC3339),
				Limit:     100,
				Page:      1,
				Order:     "desc",
			},
			expectError: true,
		},
		{
			name: "Only start time",
			params: ReadingQueryParams{
				StationID: &stationID,
				StartTime: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Limit:     100,
				Page:      1,
				Order:     "desc",
			},
			expectError: false,
		},
		{
			name: "Only end time",
			params: ReadingQueryParams{
				StationID: &stationID,
				EndTime:   now.Format(time.RFC3339),
				Limit:     100,
				Page:      1,
				Order:     "desc",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
