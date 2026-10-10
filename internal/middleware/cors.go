package middleware

import "net/http"

// CORS-заголовки ответа. allowOrigin "*" допустим, т.к. админка авторизуется через заголовок
// Authorization (Bearer), а не через cookie: браузер не прикладывает учетные данные сам.
const (
	corsAllowOrigin  = "*"
	corsAllowMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization"
	// corsExposeHeaders — заголовки ответа, которые JavaScript на другом домене может
	// прочитать. Без этого списка браузер скрывает все, кроме простых (Content-Type и т.п.),
	// и фронт не узнал бы, через сколько повторить запрос после 429 (см. RateLimit).
	corsExposeHeaders = "Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining"
	// corsMaxAge — сколько секунд браузер может не повторять preflight для того же запроса.
	corsMaxAge = "600"
)

// CORS разрешает кросс-доменные запросы к API (сайт и админ-панель обращаются
// к одному API с разных доменов) — добавляет CORS-заголовки к ответу маршрута.
//
// Preflight-запросы (OPTIONS) сюда не доходят: маршруты зарегистрированы с методом
// ("POST /api/v1/leads"), и на OPTIONS http.ServeMux сам ответил бы 405. Их обрабатывает
// Preflight, которым обернут весь роутер.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w.Header())
		next(w, r)
	}
}

// Preflight отвечает на CORS preflight-запросы (OPTIONS с заголовком
// Access-Control-Request-Method) для любого пути — до маршрутизации. Оборачивает весь
// http.ServeMux (см. cmd/server/main.go); остальные запросы передает дальше без изменений.
func Preflight(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			setCORSHeaders(w.Header())
			w.Header().Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setCORSHeaders(h http.Header) {
	h.Set("Access-Control-Allow-Origin", corsAllowOrigin)
	h.Set("Access-Control-Allow-Methods", corsAllowMethods)
	h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
	h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
}
