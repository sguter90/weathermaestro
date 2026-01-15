package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/puller/netatmo"
)

// netatmoCallbackHandler handles the OAuth2 callback from Netatmo
func (rm *RouteManager) netatmoCallbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stationIDStr := vars["stationID"]
	stationID, err := uuid.Parse(stationIDStr)
	if err != nil {
		http.Error(w, "Invalid station_id format", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Get current config
	config, err := rm.dbManager.GetStationConfig(stationID)
	if err != nil {
		http.Error(w, "Config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Exchange code for access token
	clientID, ok := config["client_id"].(string)
	if !ok {
		http.Error(w, "Invalid client_id in config", http.StatusInternalServerError)
		return
	}

	clientSecret, ok := config["client_secret"].(string)
	if !ok {
		http.Error(w, "Invalid client_secret in config", http.StatusInternalServerError)
		return
	}

	redirectURI, ok := config["redirect_uri"].(string)
	if !ok {
		http.Error(w, "Invalid redirect_uri in config", http.StatusInternalServerError)
		return
	}

	dbState, ok := config["state"].(string)
	if !ok {
		http.Error(w, "Invalid state in config", http.StatusInternalServerError)
		return
	}

	client := netatmo.NewClient(clientID, clientSecret, redirectURI)
	client.SetState(dbState)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := client.GetAccessTokenFromCode(ctx, code, state); err != nil {
		log.Printf("Failed to get access token: %v", err)
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return
	}

	// Update config with access token
	config["access_token"] = client.GetAccessToken()
	config["refresh_token"] = client.GetAccessToken()
	config["token_expiry"] = client.GetTokenExpiry().Format(time.RFC3339)

	err = rm.dbManager.SetStationConfig(stationID, config)
	if err != nil {
		log.Printf("Failed to update station config: %v", err)
		http.Error(w, "Failed to save access token", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully saved access token for station %s", stationID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Authorization successful! You can close this window.",
	})
}
