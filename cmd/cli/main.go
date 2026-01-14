package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sguter90/weathermaestro/pkg/database"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "weathermaestro",
	Short: "WeatherMaestro - Weather Station Management System",
	Long: `WeatherMaestro is a comprehensive weather station management system
that supports multiple weather station types and data sources.`,
}

func main() {
	dbManager, err := database.NewDatabaseManager()
	if err != nil {
		fmt.Printf("Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer dbManager.Close()

	ctx := context.WithValue(context.Background(), "dbManager", dbManager)
	rootCmd.SetContext(ctx)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
