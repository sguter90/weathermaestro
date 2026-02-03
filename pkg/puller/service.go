package puller

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// PullerService manages periodic data pulling from external providers
type PullerService struct {
	dbManager      *database.DatabaseManager
	pullerRegistry *PullerRegistry
	interval       time.Duration
	stopChan       chan struct{}
	stations       map[string]*models.StationData
	mu             sync.RWMutex
	ticker         *time.Ticker
}

// NewPullerService creates a new PullerService
func NewPullerService(dbManager *database.DatabaseManager, registry *PullerRegistry, interval time.Duration) *PullerService {
	return &PullerService{
		dbManager:      dbManager,
		pullerRegistry: registry,
		interval:       interval,
		stopChan:       make(chan struct{}),
		stations:       make(map[string]*models.StationData),
	}
}

// AddStation adds a station for pulling
func (ps *PullerService) AddStation(data *models.StationData) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.stations[data.ID.String()] = data
}

// Start begins the periodic pulling service
func (ps *PullerService) Start() {
	go ps.run()
	log.Println("✓ Puller service started")
}

// Stop halts the pulling service
func (ps *PullerService) Stop() {
	close(ps.stopChan)
	if ps.ticker != nil {
		ps.ticker.Stop()
	}
	log.Println("✓ Puller service stopped")
}

// run executes the pulling loop
func (ps *PullerService) run() {
	ps.ticker = time.NewTicker(ps.interval)
	defer ps.ticker.Stop()

	// Pull immediately on start
	ps.pullAllProviders()

	for {
		select {
		case <-ps.stopChan:
			return
		case <-ps.ticker.C:
			ps.pullAllProviders()
		}
	}
}

// pullAllProviders pulls data from all configured providers
func (ps *PullerService) pullAllProviders() {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, s := range ps.stations {
		// fetch latest s config from database
		s, err := ps.dbManager.LoadStation(s.ID)
		if err != nil {
			fmt.Printf("Failed to query s: %v\n", err)
			continue
		}

		p, ok := ps.pullerRegistry.Get(s.ServiceName)
		if !ok {
			log.Printf("⚠ Puller not found for provider type: %s", s.ServiceName)
			continue
		}

		ps.pullFromProvider(p, s.Config)
	}
}

// pullFromProvider pulls data from a specific provider
func (ps *PullerService) pullFromProvider(p Puller, config map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sensorReadings, _, err := p.Pull(ctx, config)
	if err != nil {
		log.Printf("❌ Error pulling from %s: %v", p.GetProviderType(), err)
		return
	}

	if len(sensorReadings) == 0 {
		log.Printf("❌ No weather data received from %s", p.GetProviderType())
		return
	}

	// Store weather data
	for _, reading := range sensorReadings {
		if err := ps.dbManager.StoreSensorReading(reading.SensorID, reading.Value, reading.DateUTC); err != nil {
			log.Printf("❌ Error storing weather data (%s, %f, %s): %v", reading.SensorID.String(), reading.Value, reading.DateUTC, err)
			return
		}
	}

	log.Printf("✓ Pulled %d Weather readings for station: %s", len(sensorReadings), p.GetProviderType())
}
