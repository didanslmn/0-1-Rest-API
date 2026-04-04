package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"toko-produk/middleware"
	"toko-produk/models"
	"toko-produk/repository"

	"github.com/jackc/pgx/v5"
)

type ProductHandler struct {
	ProductRepo *repository.ProductRepository
}

func NewProductHandler(productRepo *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{ProductRepo: productRepo}
}

// GET /products
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	filter := models.ProdyctFilter{
		Category: r.URL.Query().Get("category"),
		MaxPrice: r.URL.Query().Get("max_price"),
		MinPrice: r.URL.Query().Get("min_price"),
	}

	product, err := h.ProductRepo.GetAll(r.Context(), filter)
	if err != nil {
		http.Error(w, "error saat mengambil data produk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "data produk berhasil diambil",
		"data":    product,
	})
}

// GET /product/id
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	// r.Pathvalue("id")
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id produk tidak ditemukan", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		http.Error(w, "id produk tidak valid", http.StatusBadRequest)
		return
	}

	product, err := h.ProductRepo.GetByID(r.Context(), id)
	if err == pgx.ErrNoRows {
		http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "error saat mengambil data produk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "data produk berhasil diambil",
		"data":    product,
	})
}

// POST /product
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// ambil user id dari context
	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "user tidak terautentikasi", http.StatusUnauthorized)
		return
	}

	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "nama produk tidak boleh kosong", http.StatusBadRequest)
		return
	}

	if req.Price <= 0 {
		http.Error(w, "harga produk harus lebih besar dari 0", http.StatusBadRequest)
		return
	}

	if req.Stock < 0 {
		http.Error(w, "stock produk tidak boleh negatif", http.StatusBadRequest)
		return
	}

	product, err := h.ProductRepo.Create(r.Context(), req, claims.UserID)
	if err != nil {
		http.Error(w, "error saat membuat produk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "produk berhasil dibuat",
		"data":    product,
	})
}

// PUT /product/{id}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 {
		http.Error(w, "id tidak valid", http.StatusBadRequest)
		return
	}

	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == nil && req.Description == nil && req.Price == nil && req.Stock == nil && req.Category == nil {
		http.Error(w, "tidak ada data yang diupdate", http.StatusBadRequest)
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		http.Error(w, "nama tidak boleh kosong", http.StatusBadRequest)
		return
	}

	if req.Price != nil && *req.Price <= 0 {
		http.Error(w, "harga tidak boleh negatif", http.StatusBadRequest)
		return
	}

	product, err := h.ProductRepo.Update(r.Context(), req, id, claims.UserID)
	if err != nil {
		if _, ok := err.(*repository.NotFoundError); ok {
			http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
			return
		}
		http.Error(w, "error saat update produk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "produk berhasil diupdate",
		"data":    product,
	})
}

// DELETE /product/{id}
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 {
		http.Error(w, "id tidak valid", http.StatusBadRequest)
		return
	}

	err = h.ProductRepo.Delete(r.Context(), id, claims.UserID)
	if err != nil {
		if _, ok := err.(*repository.NotFoundError); ok {
			http.Error(w, "produk tidak ditemukan", http.StatusNotFound)
			return
		}
		http.Error(w, "error saat delete produk", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "produk berhasil dihapus",
	})
}
