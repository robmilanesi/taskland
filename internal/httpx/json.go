// Package httpx provides utilities for the api and web management
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse is the JSON body returned for error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSON encodes data as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

// WriteError writes an ErrorResponse with the given status code and message.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// WriteISE writes a generic 500 Internal Server Error response.
func WriteISE(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, "internal server error")
}

// DecodeJSON decodes the request body into dst, rejecting unknown fields and
// bodies larger than 1 MiB. On failure it writes a 400 response and returns
// false, so callers can simply return.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}
