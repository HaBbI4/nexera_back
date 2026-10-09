package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Константы
const (
	defaultCacheTTL = 15 * time.Minute
)

// ErrorResponse представляет структуру ответа с ошибкой
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeJSONResponse отправляет JSON ответ
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeErrorResponse отправляет ответ с ошибкой
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	writeJSONResponse(w, statusCode, response)
}
