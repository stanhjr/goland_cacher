package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MakeHeaders sets the specified headers and HTTP status on the provided http.ResponseWriter.
func MakeHeaders(w http.ResponseWriter, headers http.Header, status int) {
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(status)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteJSONError(w http.ResponseWriter, errorMessage string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ErrorResponse{Error: errorMessage}
	jsonResponse, err := json.Marshal(errorResponse)
	if err != nil {
		// If there's an error while encoding, just write the error message
		fmt.Fprintf(w, `{"error": "Failed to encode error response"}`)
		return
	}

	_, _ = w.Write(jsonResponse)
}
