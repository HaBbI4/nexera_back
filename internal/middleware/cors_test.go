package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexera_back/internal/middleware"
)

// newTestRouter повторяет устройство роутера из cmd/server/main.go: маршруты с методом
// в паттерне, CORS на маршруте, Preflight вокруг всего mux.
func newTestRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/leads", middleware.Chain(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}, middleware.CORS))
	mux.HandleFunc("PATCH /api/v1/admin/faq/{id}", middleware.Chain(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, middleware.CORS))
	return middleware.Preflight(mux)
}

func TestPreflight_AnswersForMethodSpecificRoutes(t *testing.T) {
	router := newTestRouter()

	for _, tc := range []struct{ path, method string }{
		{"/api/v1/leads", "POST"},
		{"/api/v1/admin/faq/1", "PATCH"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodOptions, tc.path, nil)
			r.Header.Set("Origin", "https://other.example")
			r.Header.Set("Access-Control-Request-Method", tc.method)
			r.Header.Set("Access-Control-Request-Headers", "content-type, authorization")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			if w.Code != http.StatusNoContent {
				t.Fatalf("preflight status = %d, want 204", w.Code)
			}
			if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), tc.method) {
				t.Errorf("Allow-Methods %q does not contain %s", w.Header().Get("Access-Control-Allow-Methods"), tc.method)
			}
			if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
				t.Errorf("Allow-Headers must contain Authorization, got %q", w.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}

func TestPreflight_PassesThroughRegularRequests(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/leads", nil)
	w := httptest.NewRecorder()

	newTestRouter().ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}

func TestCORS_ExposesRateLimitHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/leads", nil)
	w := httptest.NewRecorder()

	newTestRouter().ServeHTTP(w, r)

	exposed := w.Header().Get("Access-Control-Expose-Headers")
	for _, header := range []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining"} {
		if !strings.Contains(exposed, header) {
			t.Errorf("Expose-Headers %q does not contain %s", exposed, header)
		}
	}
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Allow-Origin must be set on actual responses")
	}
}
