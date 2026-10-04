package handlers

import (
	"encoding/json"
	"fmt"
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

// enableCORS включает CORS заголовки
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// corsMiddleware добавляет CORS заголовки
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// loggingMiddleware логирует HTTP запросы
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Вызываем следующий обработчик
		next(w, r)

		// Логируем запрос
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s - %v\n",
			start.Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			duration)
	}
}
