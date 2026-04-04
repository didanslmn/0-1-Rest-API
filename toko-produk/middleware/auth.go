package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"toko-produk/models"

	"github.com/golang-jwt/jwt/v5"
)

// untuk menyimpan user id di context
type contextKey string

const ClaimsKey contextKey = "claims"

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: msg,
	})
}

// auth adalah middleware yang melindungi route
// cara kerja: cek header auth -> validasi jwt -> simpan user id di context -> next handler

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ambil token dari header auth
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "token tidak ditemuan,header authorization tidak ada")
			return
		}
		// pisahkan header auth menjadi 2 bagian: Bearer dan token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "format header authorization salah, gunakan:  gunakan: Bearer <token>")
			return
		}
		// ambil token
		tokenString := parts[1]
		// validasi token
		token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (any, error) {
			// cek method signing pastikan algoritma adalah HMAC(HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "token tidak valid")
			return
		}

		// ambil claims dari token
		claims, ok := token.Claims.(*models.UserClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "token tidak valid")
			return
		}

		// simpan user id di context
		r = r.WithContext(context.WithValue(r.Context(), ClaimsKey, *claims))
		next.ServeHTTP(w, r)
	}
}

// GetClaims
func GetClaims(r *http.Request) (models.UserClaims, bool) {
	claims, ok := r.Context().Value(ClaimsKey).(models.UserClaims)
	return claims, ok
}
