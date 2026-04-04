package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"toko-produk/models"
	"toko-produk/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserRepo *repository.UserRepository
}

func NewAuthHandler(userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "nama tidak boleh kosong", http.StatusBadRequest)
		return
	}

	if !strings.Contains(req.Email, "@") {
		http.Error(w, "email tidak valid", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "password minimal 6 karakter", http.StatusBadRequest)
		return
	}

	exists, err := h.UserRepo.EmailExists(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "error saat cek email", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "email sudah terdaftar", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "error saat generate password", http.StatusInternalServerError)
		return
	}

	user := &models.User{
		Name:         strings.TrimSpace(req.Name),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
	}

	if err := h.UserRepo.CreateUser(r.Context(), user); err != nil {
		http.Error(w, "error saat membuat user", http.StatusInternalServerError)
		return
	}

	token, err := generateToken(user)
	if err != nil {
		http.Error(w, "error saat generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "user berhasil dibuat",
		"data":    models.AuthResponse{Token: token, User: user},
	})

}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email dan password tidak boleh kosong", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.FindByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "email atau password salah", http.StatusUnauthorized)
		return
	}
	// membandingkan password yang di input dengan password yang ada di database
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "email atau password salah", http.StatusUnauthorized)
		return
	}

	token, err := generateToken(user)
	if err != nil {
		http.Error(w, "error saat generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "login berhasil",
		"data":    models.AuthResponse{Token: token, User: user},
	})
}

// helper --generate token
func generateToken(user *models.User) (string, error) {
	expiryHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_HOURS"))
	if expiryHours == 0 {
		expiryHours = 24
	}

	claims := models.UserClaims{
		UserID: user.ID,
		Name:   user.Name,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expiryHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
