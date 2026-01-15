package main

import (
	"encoding/json"
	"net/http"
)

// healthHandler returns server health status
func (rm *RouteManager) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
