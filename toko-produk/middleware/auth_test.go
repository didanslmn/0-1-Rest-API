package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"toko-produk/middleware"
	"toko-produk/models"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestToken membuat jwt token valid untuk keperluan test
func generateTestToken(t *testing.T, UserID int, email, name string) string {
	t.Helper()

	os.Setenv("JWT_SECRET", "secret-test-123")

	claims := models.UserClaims{
		UserID: UserID,
		Email:  email,
		Name:   name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("secret-test-123"))
	if err != nil {
		t.Fatalf("gagal membuat token: %v", err)
	}
	return tokenString
}

func TestAuth(t *testing.T) {
	os.Setenv("JWT_SECRET", "secret-test-123")
	// handler dummy untuk test
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			t.Error("claims tidak ditemukan")
			return
		}
		if claims.UserID != 1 {
			t.Error("user id tidak valid")
			return
		}
		w.WriteHeader(http.StatusOK)

	})
	// table driven test
	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid token",
			authHeader: "Bearer " + generateTestToken(t, 1, "test@example.com", "test"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer token-palsu",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "format header salah, tanpa bearer",
			authHeader: generateTestToken(t, 1, "test@example.com", "test"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			// tambahkan test token expired
			name: "token expired",
			authHeader: "Bearer " + func() string {
				claims := jwt.MapClaims{
					"user_id": float64(1),
					"email":   "test@example.com",
					"name":    "test",
					"exp":     time.Now().Add(-time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, err := token.SignedString([]byte("secret-test-123"))
				if err != nil {
					t.Fatalf("gagal membuat token: %v", err)
				}
				return tokenString
			}(),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// buat request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			// buat response recorder
			w := httptest.NewRecorder()
			// jalankan middleware
			middleware.Auth(dummyHandler).ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}

}
