package handler

import (
	"encoding/json"
	"net/http"
)

// ResponseEnvelope wraps standard JSON API responses.
type ResponseEnvelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Total   int    `json:"total,omitempty"`
}

// WriteJSON writes a typed JSON response with status code and headers.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes an enveloped success response.
func WriteSuccess(w http.ResponseWriter, status int, data any, total int) {
	WriteJSON(w, status, ResponseEnvelope{
		Success: true,
		Data:    data,
		Total:   total,
	})
}

// WriteError writes an enveloped error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ResponseEnvelope{
		Success: false,
		Error:   message,
	})
}

// ReadJSON decodes the request body into target destination.
func ReadJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

