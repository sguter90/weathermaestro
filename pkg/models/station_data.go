package models

import "github.com/google/uuid"

type StationData struct {
	ID          uuid.UUID `json:"id"`
	PassKey     string    `json:"pass_key"`
	StationType string    `json:"station_type"`
	Model       string    `json:"model"`
	Freq        string    `json:"freq"`
	Interval    int       `json:"interval"`
}
