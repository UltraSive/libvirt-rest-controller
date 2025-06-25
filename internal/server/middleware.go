package server

import (
	"net/http"
	"os"
	"strings"

	"libvirt-controller/internal/server/utils"
)

// AuthMiddleware checks for a valid Bearer token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedToken := os.Getenv("AUTH_TOKEN")

		// If AUTH_TOKEN is not configured, proceed with the request unconditionally
		if expectedToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")

		// If AUTH_TOKEN is set, check for the Authorization header
		if authHeader == "" {
			utils.JSONErrorResponse(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer prefix and extract the token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != expectedToken {
			utils.JSONErrorResponse(w, "Invalid or missing token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed with the request
		next.ServeHTTP(w, r)
	})
}
