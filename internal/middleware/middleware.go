package middleware

import "net/http"

// Middleware оборачивает обработчик дополнительным поведением (CORS, аутентификация и т.д.).
// Один и тот же тип используется и для публичных, и для админских маршрутов — различие
// достигается набором middleware, применяемых к конкретному маршруту в cmd/server/main.go.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain применяет middleware к обработчику в порядке перечисления слева направо:
// Chain(h, A, B) эквивалентно A(B(h)) — A выполнится раньше B.
func Chain(h http.HandlerFunc, mws ...Middleware) http.HandlerFunc {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
