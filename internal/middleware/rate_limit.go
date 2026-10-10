package middleware

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
)

// RateCounter — счетчик запросов с фиксированным окном. Реализуется *redis.Client
// (IncrementWithTTL); интерфейс — чтобы middleware можно было тестировать без Redis.
type RateCounter interface {
	IncrementWithTTL(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
}

// RateLimitRule — ограничение для одного эндпоинта: не больше MaxRequests за Window с одного IP.
// Name входит в ключ счетчика, чтобы лимиты разных эндпоинтов не смешивались.
type RateLimitRule struct {
	Name        string
	MaxRequests int
	Window      time.Duration
}

// RateLimit ограничивает частоту запросов с одного IP клиента (аналог throttle в Laravel).
// Сверх лимита — 429 с заголовком Retry-After; в каждом ответе — X-RateLimit-Limit
// и X-RateLimit-Remaining.
//
// При недоступном Redis запрос пропускается (fail-open): потерять заявку клиента хуже,
// чем пропустить немного спама. Правило с MaxRequests <= 0 отключает ограничение.
func RateLimit(counter RateCounter, rule RateLimitRule, ipResolver *ClientIPResolver, log *logger.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		if rule.MaxRequests <= 0 || rule.Window <= 0 {
			return next
		}

		return func(w http.ResponseWriter, r *http.Request) {
			ip := ipResolver.ClientIP(r)
			key := "ratelimit:" + rule.Name + ":" + ip

			count, ttl, err := counter.IncrementWithTTL(r.Context(), key, rule.Window)
			if err != nil {
				log.WithError(err).WithField("rule", rule.Name).Error("rate limit check failed, request allowed")
				next(w, r)
				return
			}

			remaining := max(int64(rule.MaxRequests)-count, 0)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rule.MaxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

			if count > int64(rule.MaxRequests) {
				if ttl <= 0 {
					ttl = rule.Window
				}
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(ttl.Seconds()))))

				log.WithFields(map[string]interface{}{
					"rule": rule.Name,
					"ip":   ip,
				}).Warn("rate limit exceeded")

				httpresponse.WriteError(w, http.StatusTooManyRequests, httpresponse.TooManyRequestsMessage)
				return
			}

			next(w, r)
		}
	}
}
