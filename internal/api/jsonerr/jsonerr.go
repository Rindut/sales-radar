// Package jsonerr writes a consistent JSON error envelope for the Sales Radar API.
package jsonerr

import (
	"encoding/json"
	"net/http"
)

// Envelope is the standard API error shape.
type Envelope struct {
	Error Detail `json:"error"`
}

// Detail holds machine-oriented code and human message.
type Detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Write sends a JSON error response with the given HTTP status.
func Write(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Error: Detail{Code: code, Message: message},
	})
}
