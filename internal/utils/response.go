package utils

import (
	"encoding/json"
	"net/http"
)

// represents a HTTP response
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// sends a JSON response
func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sends a JSON response with an error
func SendError(w http.ResponseWriter, status int, message string) {
	SendJSON(w, status, Response{
		Status:  "error",
		Message: message,
	})
}

// sends a JSON response with a success result
func SendSuccess(w http.ResponseWriter, data interface{}) {
	SendJSON(w, http.StatusOK, Response{
		Status: "success",
		Data:   data,
	})
}
