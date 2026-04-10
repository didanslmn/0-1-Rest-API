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

func (h *ProductHandler) response(w http.ResponseWriter, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"success": code >= 200 && code < 300,
		"message": message,
		"data":    data,
	})
}

// helper --
func parsePagination(r *http.Request) (page, limit int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}
	return page, limit
}

// GET /products
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	filter := models.ProdyctFilter{
		Category: r.URL.Query().Get("category"),
		MaxPrice: r.URL.Query().Get("max_price"),
		MinPrice: r.URL.Query().Get("min_price"),
		Search:   r.URL.Query().Get("search"),
	}

	page, limit := parsePagination(r)

	product, err := h.ProductRepo.GetAll(r.Context(), filter, page, limit)
	if err != nil {
		h.response(w, http.StatusInternalServerError, "error saat mengambil data produk", nil)
		return
	}

	h.response(w, http.StatusOK, "data produk berhasil diambil", product)
}

// GET /prodict/me
func (h *ProductHandler) GetMyProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		h.response(w, http.StatusUnauthorized, "user tidak terautentikasi", nil)
		return
	}
	filter := models.ProdyctFilter{
		UserID:   claims.UserID,
		Category: r.URL.Query().Get("category"),
		Search:   r.URL.Query().Get("search"),
	}

	page, limit := parsePagination(r)

	product, err := h.ProductRepo.GetAll(r.Context(), filter, page, limit)
	if err != nil {
		h.response(w, http.StatusInternalServerError, "error saat mengambil data produk", nil)
		return
	}

	h.response(w, http.StatusOK, "data produk berhasil diambil", product)
}

// GET /product/id
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	// r.Pathvalue("id")
	idStr := r.PathValue("id")
	if idStr == "" {
		h.response(w, http.StatusBadRequest, "id produk tidak ditemukan", nil)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		h.response(w, http.StatusBadRequest, "id produk tidak valid", nil)
		return
	}

	product, err := h.ProductRepo.GetByID(r.Context(), id)
	if err == pgx.ErrNoRows {
		h.response(w, http.StatusNotFound, "produk tidak ditemukan", nil)
		return
	}
	if err != nil {
		h.response(w, http.StatusInternalServerError, "error saat mengambil data produk", nil)
		return
	}

	h.response(w, http.StatusOK, "data produk berhasil diambil", product)
}

// POST /product
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// ambil user id dari context
	claims, ok := middleware.GetClaims(r)
	if !ok {
		h.response(w, http.StatusUnauthorized, "user tidak terautentikasi", nil)
		return
	}

	var req models.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.response(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		h.response(w, http.StatusBadRequest, "nama produk tidak boleh kosong", nil)
		return
	}

	if req.Price <= 0 {
		h.response(w, http.StatusBadRequest, "harga produk harus lebih besar dari 0", nil)
		return
	}

	if req.Stock < 0 {
		h.response(w, http.StatusBadRequest, "stock produk tidak boleh negatif", nil)
		return
	}

	product, err := h.ProductRepo.Create(r.Context(), req, claims.UserID)
	if err != nil {
		h.response(w, http.StatusInternalServerError, "error saat membuat produk", nil)
		return
	}

	h.response(w, http.StatusCreated, "produk berhasil dibuat", product)
}

// PUT /product/{id}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 {
		h.response(w, http.StatusBadRequest, "id tidak valid", nil)
		return
	}

	var req models.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.response(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	if req.Name == nil && req.Description == nil && req.Price == nil && req.Stock == nil && req.Category == nil {
		h.response(w, http.StatusBadRequest, "tidak ada data yang diupdate", nil)
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		h.response(w, http.StatusBadRequest, "nama tidak boleh kosong", nil)
		return
	}

	if req.Price != nil && *req.Price <= 0 {
		h.response(w, http.StatusBadRequest, "harga tidak boleh negatif", nil)
		return
	}

	product, err := h.ProductRepo.Update(r.Context(), req, id, claims.UserID)
	if err != nil {
		if _, ok := err.(*repository.NotFoundError); ok {
			h.response(w, http.StatusNotFound, "produk tidak ditemukan", nil)
			return
		}
		h.response(w, http.StatusInternalServerError, "error saat update produk", nil)
		return
	}

	h.response(w, http.StatusOK, "produk berhasil diupdate", product)
}

// DELETE /product/{id}
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetClaims(r)

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 {
		h.response(w, http.StatusBadRequest, "id tidak valid", nil)
		return
	}

	err = h.ProductRepo.Delete(r.Context(), id, claims.UserID)
	if err != nil {
		if _, ok := err.(*repository.NotFoundError); ok {
			h.response(w, http.StatusNotFound, "produk tidak ditemukan", nil)
			return
		}
		h.response(w, http.StatusInternalServerError, "error saat delete produk", nil)
		return
	}

	h.response(w, http.StatusOK, "produk berhasil dihapus", nil)
}
