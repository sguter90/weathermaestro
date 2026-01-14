package main

import (
	"fmt"
	"time"

	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/models"
	"github.com/sguter90/weathermaestro/pkg/puller"
	"github.com/sguter90/weathermaestro/pkg/pusher"
)

type RegistryManager struct {
	PusherRegistry *pusher.Registry
	PullerRegistry *puller.PullerRegistry
	PullerService  *puller.PullerService
}

func InitRegistryManager(dbManager *database.DatabaseManager, stations []models.StationData) *RegistryManager {
	pusherRegistry := pusher.NewRegistry()
	pullerRegistry := puller.NewPullerRegistry()

	// Register pushers and pullers based on loaded stations
	for _, station := range stations {
		if station.Mode == "push" {
			fmt.Printf("Registering pusher: %s (%s)\n", station.Model, station.ServiceName)
			registerPusher(pusherRegistry, station.ServiceName)
		} else if station.Mode == "pull" {
			fmt.Printf("Registering puller: %s (%s)\n", station.Model, station.ServiceName)
			registerPuller(pullerRegistry, station.ServiceName, dbManager)
		}
	}

	// Initialize puller service
	pullerService := puller.NewPullerService(dbManager, pullerRegistry, 1*time.Minute)

	// Add stations to puller service
	for _, station := range stations {
		if station.Mode == "pull" {
			config := station.Config
			stringConfig := make(map[string]string)
			for k, v := range config {
				stringConfig[k] = fmt.Sprintf("%v", v)
			}
			pullerService.AddStationConfig(station.ServiceName, stringConfig)
		}
	}

	return &RegistryManager{
		PusherRegistry: pusherRegistry,
		PullerRegistry: pullerRegistry,
		PullerService:  pullerService,
	}
}
