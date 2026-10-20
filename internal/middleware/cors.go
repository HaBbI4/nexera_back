package middleware

import (
	"net/http"
	"strings"
)

const (
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization"
	// corsExposeHeaders — заголовки ответа, которые JavaScript на другом домене может
	// прочитать. Без этого списка браузер скрывает все, кроме простых (Content-Type и т.п.),
	// и фронт не узнал бы, через сколько повторить запрос после 429 (см. RateLimit).
	corsExposeHeaders = "Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining"
	// corsMaxAge — сколько секунд браузер может не повторять preflight для того же запроса.
	corsMaxAge = "600"
)

// corsAllowedOrigins — список Origin, которым отвечаем Access-Control-Allow-Origin.
// "*" или пустой список (до InitCORS) — разрешить любой Origin (как раньше).
// Задаётся из main через InitCORS / CORS_ALLOWED_ORIGINS.
var corsAllowedOrigins = []string{"*"}

// InitCORS настраивает разрешённые Origin для публичного фронтенда (nexera_front).
// Передайте список вида http://localhost:3000,https://site.example.
// Элемент "*" сохраняет прежнее поведение «разрешить всё».
func InitCORS(origins []string) {
	cleaned := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			cleaned = append(cleaned, o)
		}
	}
	if len(cleaned) == 0 {
		corsAllowedOrigins = []string{"*"}
		return
	}
	corsAllowedOrigins = cleaned
}

// CORS разрешает кросс-доменные запросы к API (сайт и админ-панель обращаются
// к одному API с разных доменов) — добавляет CORS-заголовки к ответу маршрута.
//
// Preflight-запросы (OPTIONS) сюда не доходят: маршруты зарегистрированы с методом
// ("POST /api/v1/leads"), и на OPTIONS http.ServeMux сам ответил бы 405. Их обрабатывает
// Preflight, которым обернут весь роутер.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w.Header(), r.Header.Get("Origin"))
		next(w, r)
	}
}

// Preflight отвечает на CORS preflight-запросы (OPTIONS с заголовком
// Access-Control-Request-Method) для любого пути — до маршрутизации. Оборачивает весь
// http.ServeMux (см. cmd/server/main.go); остальные запросы передает дальше без изменений.
func Preflight(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			setCORSHeaders(w.Header(), r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setCORSHeaders(h http.Header, requestOrigin string) {
	h.Set("Access-Control-Allow-Origin", resolveAllowOrigin(requestOrigin))
	h.Set("Access-Control-Allow-Methods", corsAllowMethods)
	h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
	// Когда Origin отражается явно (не "*"), Vary нужен для корректного кеширования прокси.
	if requestOrigin != "" {
		h.Add("Vary", "Origin")
	}
}

func resolveAllowOrigin(requestOrigin string) string {
	for _, allowed := range corsAllowedOrigins {
		if allowed == "*" {
			return "*"
		}
	}
	if requestOrigin == "" {
		// Нет Origin (curl/same-origin) — отдаём первый явный origin из списка.
		return corsAllowedOrigins[0]
	}
	for _, allowed := range corsAllowedOrigins {
		if strings.EqualFold(allowed, requestOrigin) {
			return requestOrigin
		}
	}
	// Неразрешённый Origin: не ставим чужой домен в Allow-Origin.
	return ""
}
