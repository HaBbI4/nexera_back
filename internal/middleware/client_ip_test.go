package middleware_test

import (
	"net/http/httptest"
	"testing"

	"nexera_back/internal/middleware"
)

func TestClientIPResolver_ClientIP(t *testing.T) {
	resolver, err := middleware.NewClientIPResolver([]string{"127.0.0.1", "172.16.0.0/12"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		xff        []string
		want       string
	}{
		{
			name:       "direct connection without proxy",
			remoteAddr: "203.0.113.5:51234",
			want:       "203.0.113.5",
		},
		{
			name:       "untrusted client cannot spoof X-Forwarded-For",
			remoteAddr: "203.0.113.5:51234",
			xff:        []string{"1.1.1.1"},
			want:       "203.0.113.5",
		},
		{
			name:       "behind trusted nginx",
			remoteAddr: "127.0.0.1:40000",
			xff:        []string{"203.0.113.5"},
			want:       "203.0.113.5",
		},
		{
			name:       "client-supplied prefix in header is ignored",
			remoteAddr: "127.0.0.1:40000",
			xff:        []string{"1.1.1.1, 203.0.113.5"},
			want:       "203.0.113.5",
		},
		{
			name:       "chain of trusted proxies (nginx -> docker network)",
			remoteAddr: "172.18.0.1:40000",
			xff:        []string{"203.0.113.5, 127.0.0.1"},
			want:       "203.0.113.5",
		},
		{
			name:       "multiple header lines are joined in order",
			remoteAddr: "127.0.0.1:40000",
			xff:        []string{"1.1.1.1", "203.0.113.5"},
			want:       "203.0.113.5",
		},
		{
			name:       "trusted proxy without header",
			remoteAddr: "127.0.0.1:40000",
			want:       "127.0.0.1",
		},
		{
			name:       "garbage in header stops at last valid hop",
			remoteAddr: "127.0.0.1:40000",
			xff:        []string{"not-an-ip"},
			want:       "127.0.0.1",
		},
		{
			name:       "IPv4-mapped IPv6 remote address",
			remoteAddr: "[::ffff:127.0.0.1]:40000",
			xff:        []string{"203.0.113.5"},
			want:       "203.0.113.5",
		},
		{
			name:       "IPv6 client",
			remoteAddr: "127.0.0.1:40000",
			xff:        []string{"2001:db8::1"},
			want:       "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/leads", nil)
			r.RemoteAddr = tt.remoteAddr
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}

			if got := resolver.ClientIP(r); got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientIPResolver_NoTrustedProxies(t *testing.T) {
	resolver, err := middleware.NewClientIPResolver(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := httptest.NewRequest("POST", "/api/v1/leads", nil)
	r.RemoteAddr = "127.0.0.1:40000"
	r.Header.Set("X-Forwarded-For", "203.0.113.5")

	if got := resolver.ClientIP(r); got != "127.0.0.1" {
		t.Errorf("without trusted proxies header must be ignored, got %q", got)
	}
}

func TestNewClientIPResolver_Invalid(t *testing.T) {
	for _, item := range []string{"localhost", "10.0.0.0/33", "1.2.3"} {
		if _, err := middleware.NewClientIPResolver([]string{item}); err == nil {
			t.Errorf("expected error for %q", item)
		}
	}
}
