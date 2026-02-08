package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/sguter90/weathermaestro/pkg/models"
	"github.com/spf13/cobra"
)

var stationCmd = &cobra.Command{
	Use:   "station",
	Short: "Manage weather stations",
	Long:  `Add, list, and manage weather stations.`,
}

var stationAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new weather station",
	Long:  `Interactively add a new weather station to the system.`,
	RunE:  runStationAdd,
}

var stationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all weather stations",
	Long:  `Display all registered weather stations.`,
	RunE:  runStationList,
}

var stationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a weather station",
	Long:  `Delete a registered weather station from the system.`,
	RunE:  runStationDelete,
}

func init() {
	rootCmd.AddCommand(stationCmd)
	stationCmd.AddCommand(stationAddCmd)
	stationCmd.AddCommand(stationListCmd)
	stationCmd.AddCommand(stationDeleteCmd)
}

func runStationAdd(cmd *cobra.Command, args []string) error {
	dbManager := cmd.Context().Value("dbManager").(*database.DatabaseManager)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Add New Weather Station")
	fmt.Println(strings.Repeat("=", 60))

	// Mode selection
	fmt.Print("Mode (push/pull) [push]: ")
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "push"
	}
	if mode != "push" && mode != "pull" {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	// Service name
	fmt.Print("Service name (ecowitt/netatmo/ambient/weatherflow): ")
	serviceName, _ := reader.ReadString('\n')
	serviceName = strings.TrimSpace(serviceName)

	// Pass key
	fmt.Print("Pass key: ")
	passKey, _ := reader.ReadString('\n')
	passKey = strings.TrimSpace(passKey)

	// Model
	fmt.Print("Model: ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)

	// Frequency (MHz - for station hardware)
	fmt.Print("Frequency (MHz) [868]: ")
	freqStr, _ := reader.ReadString('\n')
	freqStr = strings.TrimSpace(freqStr)

	// Create station first
	station := &models.StationData{
		ID:          uuid.New(),
		PassKey:     passKey,
		StationType: serviceName,
		Model:       model,
		Freq:        freqStr,
		Mode:        mode,
		ServiceName: serviceName,
		Config:      map[string]interface{}{},
	}

	// Save to database
	if err := dbManager.SaveStation(station); err != nil {
		return fmt.Errorf("failed to save station: %w", err)
	}

	fmt.Printf("\n✓ Station created with ID: %s\n", station.ID)

	// Collect configuration based on service
	collector := NewServiceConfigCollector(reader, dbManager)
	config := collector.Collect(serviceName, mode, station.ID)
	station.Config = config

	// Update station with config
	if err := dbManager.SetStationConfig(station.ID, config); err != nil {
		return fmt.Errorf("failed to update station config: %w", err)
	}

	fmt.Println("\n✓ Station configured successfully!")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	return nil
}

func runStationDelete(cmd *cobra.Command, args []string) error {
	dbManager := cmd.Context().Value("dbManager").(*database.DatabaseManager)
	reader := bufio.NewReader(os.Stdin)

	// Get all stations
	stations, err := dbManager.GetStationsData()
	if err != nil {
		log.Printf("Failed to fetch stations: %v", err)
		return err
	}

	if len(stations) == 0 {
		fmt.Println("No stations registered yet.")
		return nil
	}

	// Display stations
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Select Station to Delete")
	fmt.Println(strings.Repeat("=", 80))

	for i, station := range stations {
		fmt.Printf("[%d] %s (%s) - Last Updated: %s\n",
			i+1,
			station.PassKey,
			station.StationType,
			station.UpdatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	// Get selection
	fmt.Print("\nEnter station number to delete (0 to cancel): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selection int
	_, err = fmt.Sscanf(input, "%d", &selection)
	if err != nil || selection < 0 || selection > len(stations) {
		fmt.Println("Invalid selection.")
		return nil
	}

	if selection == 0 {
		fmt.Println("Cancelled.")
		return nil
	}

	selectedStation := stations[selection-1]

	// Confirmation
	fmt.Printf("\n⚠️  Are you sure you want to delete station '%s'? (yes/no): ", selectedStation.PassKey)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" && confirm != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete station
	if err := dbManager.DeleteStation(selectedStation.ID); err != nil {
		log.Printf("Failed to delete station: %v", err)
		return fmt.Errorf("failed to delete station: %w", err)
	}

	fmt.Printf("\n✓ Station '%s' deleted successfully!\n", selectedStation.PassKey)
	fmt.Println(strings.Repeat("=", 80) + "\n")

	return nil
}

func runStationList(cmd *cobra.Command, args []string) error {
	dbManager := cmd.Context().Value("dbManager").(*database.DatabaseManager)

	stations, err := dbManager.GetStationsData()
	if err != nil {
		log.Printf("Failed to fetch station: %v", err)
		return err
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("Registered Weather Stations")
	fmt.Println(strings.Repeat("=", 80))

	count := 0
	for _, station := range stations {
		count++
		fmt.Printf("\n[%d] %s\n", count, station.PassKey)
		fmt.Printf("    ID: %s\n", station.ID)
		fmt.Printf("    Type: %s\n", station.StationType)
		fmt.Printf("    Model: %s\n", station.Model)
		fmt.Printf("    Mode: %s\n", station.Mode)
		fmt.Printf("    Service: %s\n", station.ServiceName)
		fmt.Printf("    Frequency: %s\n", station.Freq)
		fmt.Printf("    Last Updated: %s\n", station.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	if count == 0 {
		fmt.Println("No stations registered yet.")
	}

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")

	return nil
}
