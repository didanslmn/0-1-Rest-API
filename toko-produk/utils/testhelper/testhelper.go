package testhelper

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"toko-produk/middleware"
	"toko-produk/models"
)

// newRequest untuk membuat http.Request untuk keperluan testing

func NewRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()

	var req *http.Request

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("gagal marshal body:%v", err)
		}

		req = httptest.NewRequest(method, path, bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	return req
}

// NewAuthRequest untuk membuat http.Request dengan context yang sudah terisi claims
// dipakai untuk test protectd handler tanpa harus generate token sungguhan
func NewAuthRequest(t *testing.T, method, path string, body any, claims models.UserClaims) *http.Request {
	t.Helper()

	req := NewRequest(t, method, path, body)
	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, claims)
	return req.WithContext(ctx)
}

// ParseResponse mem-parse JSON response body ke map
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("gagal decode response: %v", err)
	}
	return result
}

// AssertStatus mengecek status code response
func AssertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("expected status %d, got %d", want, got)
	}
}

// AssertSuccess mengecek field"success" di file json
func AssertSuccess(t *testing.T, body map[string]any, want bool) {
	t.Helper()
	val, ok := body["success"].(bool)
	if !ok {
		t.Errorf("field success tidak ditemukan")
	}
	if val != want {
		t.Errorf("field success: expected %v, got %v", want, val)
	}
}

// AssertMessage mengecek field message
func AssertMessage(t *testing.T, body map[string]any, want string) {
	t.Helper()
	val, ok := body["message"].(string)
	if !ok {
		t.Errorf("field message tidak ditemukan")
	}
	if val != want {
		t.Errorf("field message: expected %v, got %v", want, val)
	}
}
