package utils

import (
	"encoding/json"
	"net/http"
)

// JSONResponse is a helper for sending JSON responses.
func JSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// JSONErrorResponse is a helper for sending error responses.
func JSONErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	JSONResponse(w, map[string]string{"error": message}, statusCode)
}
