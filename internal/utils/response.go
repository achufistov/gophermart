package utils

import (
	"encoding/json"
	"net/http"
)

// Response представляет структуру HTTP-ответа
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// SendJSON отправляет JSON-ответ
func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// SendError отправляет JSON-ответ с ошибкой
func SendError(w http.ResponseWriter, status int, message string) {
	SendJSON(w, status, Response{
		Status:  "error",
		Message: message,
	})
}

// SendSuccess отправляет JSON-ответ с успешным результатом
func SendSuccess(w http.ResponseWriter, data interface{}) {
	SendJSON(w, http.StatusOK, Response{
		Status: "success",
		Data:   data,
	})
}
