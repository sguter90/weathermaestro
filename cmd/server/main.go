package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/internal/api"
	"github.com/sguter90/weathermaestro/internal/database"
	"github.com/sguter90/weathermaestro/pkg/parser"
	"github.com/sguter90/weathermaestro/pkg/parser/ecowitt"
)

func main() {
	// Initialize database
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize parser registry
	registry := parser.NewRegistry()
	registry.Register(&ecowitt.Parser{})
	// registry.Register(&ambient.Parser{})
	// registry.Register(&weatherflow.Parser{})

	// Setup router
	r := mux.NewRouter()
	api.SetupRoutes(r, db, registry)

	// Start server
	log.Println("Server starting on :8059")
	log.Fatal(http.ListenAndServe(":8059", r))
}
