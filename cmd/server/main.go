package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/parser"
	"github.com/sguter90/weathermaestro/pkg/parser/ecowitt"
)

func main() {
	// Initialize database
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := initDatabase(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize parser registry
	registry := parser.NewRegistry()
	registry.Register(&ecowitt.Parser{})
	// registry.Register(&ambient.Parser{})
	// registry.Register(&weatherflow.Parser{})

	// Setup router
	r := mux.NewRouter()
	setupRoutes(r, db, registry)

	// Get server port from environment
	port := getEnv("SERVER_PORT", "8059")
	addr := ":" + port

	// Start server
	server := &http.Server{
		Handler:      r,
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Starting WeatherMaestro server on %s...", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
