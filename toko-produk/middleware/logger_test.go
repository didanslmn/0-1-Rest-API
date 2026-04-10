package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"toko-produk/middleware"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "client error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "mencatat request error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "status default 200 tanpa writeeader explisit",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ok"))
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			middleware.Logger(tt.handler).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
