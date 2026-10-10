// Package httpresponse содержит общие хелперы формирования JSON-ответов API.
// Используется всеми пакетами с HTTP-обработчиками (internal/handlers/admin,
// internal/handlers/public и т.д.), чтобы не копировать writeJSON/writeError в каждый.
package httpresponse

import (
	"encoding/json"
	"net/http"
)

// TooManyRequestsMessage — текст ответа 429 для публичных форм. Один и тот же и при
// превышении лимита запросов с IP, и при повторной заявке/отзыве с тем же телефоном:
// клиенту (и боту) не нужно знать, какая именно проверка сработала.
const TooManyRequestsMessage = "too many requests, please try again later"

// ErrorResponse представляет единую структуру ответа с ошибкой для всех обработчиков API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteJSON отправляет JSON ответ.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// WriteError отправляет ответ с ошибкой.
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}
