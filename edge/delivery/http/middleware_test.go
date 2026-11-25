package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCIDRFilter(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name     string
		allowed  []string
		remote   string
		wantCode int
	}{
		{
			name:     "пустой список — доступ всем",
			allowed:  nil,
			remote:   "8.8.8.8:1234",
			wantCode: http.StatusOK,
		},
		{
			name:     "localhost разрешён",
			allowed:  []string{"127.0.0.1/32"},
			remote:   "127.0.0.1:5555",
			wantCode: http.StatusOK,
		},
		{
			name:     "LAN разрешён",
			allowed:  []string{"192.168.0.0/16"},
			remote:   "192.168.1.42:5555",
			wantCode: http.StatusOK,
		},
		{
			name:     "внешний IP запрещён",
			allowed:  []string{"127.0.0.1/32", "192.168.0.0/16"},
			remote:   "8.8.8.8:5555",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "некорректный RemoteAddr — запрещён",
			allowed:  []string{"127.0.0.1/32"},
			remote:   "garbage",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "некорректный CIDR пропускается, но валидный работает",
			allowed:  []string{"not-a-cidr", "10.0.0.0/8"},
			remote:   "10.1.2.3:80",
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := cidrFilter(parseCIDRs(tt.allowed), okHandler)
			req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
			req.RemoteAddr = tt.remote
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestCIDRFilter_IgnoresXForwardedFor(t *testing.T) {
	// Клиент подделывает X-Forwarded-For, но фильтр должен смотреть на RemoteAddr.
	h := cidrFilter(parseCIDRs([]string{"127.0.0.1/32"}), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 (X-Forwarded-For не должен влиять)", rec.Code)
	}
}
