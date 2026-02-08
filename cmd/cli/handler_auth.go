package main

import (
	"encoding/json"
	"net/http"
	"time"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success   bool      `json:"success"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	User      UserInfo  `json:"user,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (rm *RouteManager) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Validate credentials
	user, err := rm.dbManager.ValidateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Invalid username or password",
		})
		return
	}

	// Generate JWT token
	token, expiresAt, err := GenerateJWT(user)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success:   true,
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
		},
	})
}

func (rm *RouteManager) handleLogout(w http.ResponseWriter, r *http.Request) {
	// With JWT, logout is handled client-side by removing the token
	// Optionally, you could implement a token blacklist here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (rm *RouteManager) handleMe(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UserInfo{
		ID:       user.ID.String(),
		Username: user.Username,
	})
}

func (rm *RouteManager) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate new token
	token, expiresAt, err := GenerateJWT(user)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(LoginResponse{
			Success: false,
			Message: "Failed to generate token",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Success:   true,
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       user.ID.String(),
			Username: user.Username,
		},
	})
}
