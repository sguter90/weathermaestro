package main

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// corsMiddleware handles CORS headers
func (rm *RouteManager) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment variable
		allowedOriginsEnv := getEnv("SERVER_ALLOWED_ORIGINS", "")
		var allowedOrigins []string

		if allowedOriginsEnv != "" {
			// Split comma-separated origins
			allowedOrigins = strings.Split(allowedOriginsEnv, ",")
			// Trim whitespace from each origin
			for i, origin := range allowedOrigins {
				allowedOrigins[i] = strings.TrimSpace(origin)
			}
		} else {
			// Default fallback if not configured
			allowedOrigins = []string{
				"http://localhost:5173",
				"http://localhost:3000",
			}
		}

		// Check if origin is allowed
		origin := r.Header.Get("Origin")
		if origin != "" {
			isAllowed := false
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				log.Printf("Origin '%s' is not within allowed origins: %s", origin, strings.Join(allowedOrigins, ", "))
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// contextMiddleware adds database context to requests
func (rm *RouteManager) contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "db", rm.dbManager.GetDB())
		ctx = context.WithValue(ctx, "dbManager", rm.dbManager)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
