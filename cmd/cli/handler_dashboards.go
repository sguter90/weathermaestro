package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sguter90/weathermaestro/pkg/models"
)

// Public endpoint - no auth required
func (rm *RouteManager) handleGetPublicDashboards(w http.ResponseWriter, r *http.Request) {
	dashboards, err := rm.dbManager.GetDashboards(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve dashboards", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboards)
}

// Public endpoint - get default dashboard
func (rm *RouteManager) handleGetDefaultDashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := rm.dbManager.GetDefaultDashboard(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve default dashboard", http.StatusInternalServerError)
		return
	}

	if dashboard == nil {
		http.Error(w, "No default dashboard configured", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Public endpoint - get single dashboard
func (rm *RouteManager) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid dashboard ID", http.StatusBadRequest)
		return
	}

	dashboard, err := rm.dbManager.GetDashboard(r.Context(), dashboardID)
	if err != nil {
		http.Error(w, "Dashboard not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Protected endpoint - requires auth
func (rm *RouteManager) handleCreateDashboard(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r.Context()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := rm.dbManager.CreateDashboard(r.Context(), &dashboard); err != nil {
		http.Error(w, "Failed to create dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dashboard)
}

// Protected endpoint - requires auth
func (rm *RouteManager) handleUpdateDashboard(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r.Context()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid dashboard ID", http.StatusBadRequest)
		return
	}

	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dashboard.ID = dashboardID

	if err := rm.dbManager.UpdateDashboard(r.Context(), &dashboard); err != nil {
		fmt.Printf("Failed to update dashboard: %v\n", err)
		http.Error(w, "Failed to update dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Protected endpoint - requires auth
func (rm *RouteManager) handleDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r.Context()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	dashboardID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid dashboard ID", http.StatusBadRequest)
		return
	}

	if err := rm.dbManager.DeleteDashboard(r.Context(), dashboardID); err != nil {
		fmt.Printf("Failed to delete dashboard: %v\n", err)
		http.Error(w, "Failed to delete dashboard", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
