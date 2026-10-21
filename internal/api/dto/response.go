package dto

import (
	"encoding/json"
	"net/http"
)

// ResponseEnvelope represents the standard, uniform API response envelope.
type ResponseEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents structured error details.
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response with appropriate status code and headers.
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := ResponseEnvelope{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// Error writes a structured JSON error response.
func Error(w http.ResponseWriter, statusCode int, errCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := ResponseEnvelope{
		Success: false,
		Error: &APIError{
			Code:    errCode,
			Message: message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
