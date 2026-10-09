package middleware

import (
	"net/http"
	"time"

	"nexera_back/internal/logger"
)

// statusRecorder оборачивает http.ResponseWriter, чтобы запомнить код ответа для логирования —
// сам http.ResponseWriter его нигде не хранит.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Logging логирует каждый запрос через структурированный логгер приложения:
// метод, путь, код ответа и длительность обработки.
func Logging(log *logger.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next(rec, r)

			log.WithFields(map[string]interface{}{
				"method":   r.Method,
				"path":     r.URL.Path,
				"status":   rec.status,
				"duration": time.Since(start).String(),
			}).Info("http request")
		}
	}
}
