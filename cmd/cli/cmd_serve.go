package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/puller"
	"github.com/sguter90/weathermaestro/pkg/puller/netatmo"
	"github.com/sguter90/weathermaestro/pkg/pusher"
	"github.com/sguter90/weathermaestro/pkg/pusher/ecowitt"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the WeatherMaestro server",
	Long:  `Start the WeatherMaestro server to receive and manage weather data.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" || jwtSecret == "change_me_in_production" {
		return errors.New("JWT_SECRET environment variable is not set or has an invalid value")
	}

	dbManager := cmd.Context().Value("dbManager").(*database.DatabaseManager)

	// Run migrations
	if err := dbManager.Init(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Load stations from database
	stations, err := dbManager.LoadStations()
	if err != nil {
		return fmt.Errorf("failed to load stations from database: %w", err)
	}

	registryManager := InitRegistryManager(dbManager, stations)
	pullerService := registryManager.PullerService
	pullerService.Start()

	// Setup Router
	routeManager := NewRouteManager(dbManager, registryManager)
	routeManager.Setup()

	// Get server port
	port := getEnv("SERVER_PORT", "8059")
	addr := ":" + port

	// Start server
	server := &http.Server{
		Handler:      routeManager.Router,
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal received")

		pullerService.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting WeatherMaestro server on %s...", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func registerPusher(registry *pusher.Registry, serviceName string) {
	switch serviceName {
	case "ecowitt":
		registry.Register(&ecowitt.Pusher{})
		// case "ambient":
		//     PusherRegistry.Register(&ambient.Pusher{})
		// case "weatherflow":
		//     PusherRegistry.Register(&weatherflow.Pusher{})
	}
}

func registerPuller(registry *puller.PullerRegistry, serviceName string, dbManager *database.DatabaseManager) {
	switch serviceName {
	case "netatmo":
		registry.Register(netatmo.NewPuller(dbManager))
	}
}
