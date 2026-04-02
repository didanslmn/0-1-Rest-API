package handler

import (
	"book-api/models"
	"book-api/repository"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type BookHandler struct {
	Repo *repository.BookRepository
}

func NewBookHandler(repo *repository.BookRepository) *BookHandler {
	return &BookHandler{Repo: repo}
}

// helper
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, Response{Success: false, Message: msg})
}
func extractID(path string) (int, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		return 0, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// handler
func (h *BookHandler) HandleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getAll(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method tidak didukung")
	}
}

func (h *BookHandler) HandleBookByID(w http.ResponseWriter, r *http.Request) {
	// Tangkap path khusus sebelum parsing ID
	switch r.URL.Path {
	case "/books/stats":
		h.stats(w, r)
		return
	}

	id, ok := extractID(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getByID(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method tidak didukung")
	}
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

func (h *BookHandler) getAll(w http.ResponseWriter, r *http.Request) {
	filter := models.BookFilter{
		Author: r.URL.Query().Get("author"),
		Genre:  r.URL.Query().Get("genre"),
	}

	books, err := h.Repo.GetAll(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data buku")
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: books})
}

func (h *BookHandler) getByID(w http.ResponseWriter, r *http.Request, id int) {
	book, err := h.Repo.GetByID(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "buku tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data buku")
		return
	}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: book})
}

func (h *BookHandler) create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	// Validasi field wajib
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "field 'title' wajib diisi")
		return
	}
	if strings.TrimSpace(req.Author) == "" {
		writeError(w, http.StatusBadRequest, "field 'author' wajib diisi")
		return
	}
	if req.Price < 0 {
		writeError(w, http.StatusBadRequest, "field 'price' tidak boleh negatif")
		return
	}
	if req.Stock < 0 {
		writeError(w, http.StatusBadRequest, "field 'stock' tidak boleh negatif")
		return
	}
	if req.Genre == "" {
		req.Genre = "unknown"
	}

	book, err := h.Repo.Create(req)
	if err != nil {
		if ve, ok := err.(*repository.ValidationError); ok {
			writeError(w, http.StatusBadRequest, ve.Msg)
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal menyimpan buku")
		return
	}

	writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Message: "buku berhasil ditambahkan",
		Data:    book,
	})
}

func (h *BookHandler) update(w http.ResponseWriter, r *http.Request, id int) {
	var req models.UpdateBookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}
	if req.Title == nil && req.Author == nil && req.Genre == nil &&
		req.Price == nil && req.Stock == nil && req.Published == nil {
		writeError(w, http.StatusBadRequest, "kirim minimal satu field untuk diupdate")
		return
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		writeError(w, http.StatusBadRequest, "field 'title' tidak boleh kosong")
		return
	}
	if req.Price != nil && *req.Price < 0 {
		writeError(w, http.StatusBadRequest, "field 'price' tidak boleh negatif")
		return
	}
	if req.Stock != nil && *req.Stock < 0 {
		writeError(w, http.StatusBadRequest, "field 'stock' tidak boleh negatif")
		return
	}

	book, err := h.Repo.Update(id, req)
	if err != nil {
		if ve, ok := err.(*repository.ValidationError); ok {
			writeError(w, http.StatusBadRequest, ve.Msg)
			return
		}
		if _, ok := err.(*repository.NotFoundError); ok {
			writeError(w, http.StatusNotFound, "buku tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal mengupdate buku")
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "buku berhasil diupdate",
		Data:    book,
	})
}

func (h *BookHandler) delete(w http.ResponseWriter, r *http.Request, id int) {
	err := h.Repo.Delete(id)
	if err != nil {
		if _, ok := err.(*repository.NotFoundError); ok {
			writeError(w, http.StatusNotFound, "buku tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal menghapus buku")
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "buku berhasil dihapus",
	})
}

func (h *BookHandler) stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method tidak didukung")
		return
	}
	stats, err := h.Repo.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil statistik")
		return
	}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}
