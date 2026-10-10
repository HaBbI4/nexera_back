package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/middleware"
)

// fakeRateCounter — счетчик в памяти с фиксированным TTL; err имитирует недоступный Redis.
type fakeRateCounter struct {
	counts map[string]int64
	err    error
}

func newFakeRateCounter() *fakeRateCounter {
	return &fakeRateCounter{counts: make(map[string]int64)}
}

func (f *fakeRateCounter) IncrementWithTTL(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	f.counts[key]++
	return f.counts[key], window, nil
}

func testLogger() *logger.Logger {
	return logger.New(&config.LoggerConfig{Level: "error", Format: "text"})
}

func newRateLimitedHandler(t *testing.T, counter middleware.RateCounter, maxRequests int) (http.HandlerFunc, *int) {
	t.Helper()

	resolver, err := middleware.NewClientIPResolver(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := 0
	handler := middleware.Chain(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusCreated)
	}, middleware.RateLimit(counter, middleware.RateLimitRule{
		Name:        "leads",
		MaxRequests: maxRequests,
		Window:      10 * time.Minute,
	}, resolver, testLogger()))

	return handler, &calls
}

func doRequest(handler http.HandlerFunc, remoteAddr string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/v1/leads", nil)
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func TestRateLimit_BlocksAfterLimit(t *testing.T) {
	handler, calls := newRateLimitedHandler(t, newFakeRateCounter(), 2)

	for i := 0; i < 2; i++ {
		if w := doRequest(handler, "203.0.113.5:1000"); w.Code != http.StatusCreated {
			t.Fatalf("request %d: got status %d, want 201", i+1, w.Code)
		}
	}

	w := doRequest(handler, "203.0.113.5:1000")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d, want 429", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "600" {
		t.Errorf("Retry-After = %q, want 600", got)
	}
	if got := w.Header().Get("X-RateLimit-Remaining"); got != "0" {
		t.Errorf("X-RateLimit-Remaining = %q, want 0", got)
	}
	if *calls != 2 {
		t.Errorf("handler called %d times, want 2", *calls)
	}
}

func TestRateLimit_SeparateCountersPerIP(t *testing.T) {
	handler, _ := newRateLimitedHandler(t, newFakeRateCounter(), 1)

	doRequest(handler, "203.0.113.5:1000")
	if w := doRequest(handler, "203.0.113.6:1000"); w.Code != http.StatusCreated {
		t.Fatalf("other IP must have its own limit, got status %d", w.Code)
	}
}

func TestRateLimit_FailOpenWhenCounterUnavailable(t *testing.T) {
	counter := newFakeRateCounter()
	counter.err = errors.New("redis is down")
	handler, calls := newRateLimitedHandler(t, counter, 1)

	for i := 0; i < 3; i++ {
		if w := doRequest(handler, "203.0.113.5:1000"); w.Code != http.StatusCreated {
			t.Fatalf("request %d: got status %d, want 201", i+1, w.Code)
		}
	}
	if *calls != 3 {
		t.Errorf("handler called %d times, want 3", *calls)
	}
}

func TestRateLimit_DisabledWithZeroMax(t *testing.T) {
	counter := newFakeRateCounter()
	handler, _ := newRateLimitedHandler(t, counter, 0)

	for i := 0; i < 10; i++ {
		if w := doRequest(handler, "203.0.113.5:1000"); w.Code != http.StatusCreated {
			t.Fatalf("request %d: got status %d, want 201", i+1, w.Code)
		}
	}
	if len(counter.counts) != 0 {
		t.Error("disabled limit must not touch the counter")
	}
}
