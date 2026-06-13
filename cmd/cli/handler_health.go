package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// healthHandler returns server health status
func (rm *RouteManager) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
}
