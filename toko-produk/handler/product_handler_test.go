package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"toko-produk/handler"
	"toko-produk/models"
	"toko-produk/repository"
	"toko-produk/utils/testhelper"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

func newProductHandler(t *testing.T) (*handler.ProductHandler, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("gagal membuat pgxmock: %v", err)
	}
	t.Cleanup(func() { mock.Close() })
	repo := repository.NewProductRepository(mock)
	return handler.NewProductHandler(repo), mock
}

func TestGetByID(t *testing.T) {
	tests := []struct {
		name       string
		pathID     string
		mockFn     func(pgxmock.PgxPoolIface)
		wantStatus int
		wantOK     bool
	}{
		{
			name:   "produk ditemukan",
			pathID: "1",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "description", "price", "stock",
					"category", "created_by", "created_at", "updated_at",
				}).AddRow(1, "Laptop", "desc", 10000000.0, 5,
					"elektronik", 1, "2024-01-01", "2024-01-01")
				mock.ExpectQuery(`SELECT id, name`).
					WithArgs(1).WillReturnRows(rows)
			},
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:   "produk tidak ditemukan",
			pathID: "999",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, name`).
					WithArgs(999).
					WillReturnError(pgx.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantOK:     false,
		},
		{
			name:       "ID bukan angka",
			pathID:     "abc",
			mockFn:     func(mock pgxmock.PgxPoolIface) {},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newProductHandler(t)
			tt.mockFn(mock)

			// Gunakan httptest.NewRequest dengan SetPathValue untuk simulasi routing Go 1.22
			req := httptest.NewRequest(http.MethodGet, "/products/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			h.GetProductByID(w, req)

			testhelper.AssertStatus(t, w.Code, tt.wantStatus)
			body := testhelper.ParseResponse(t, w)
			testhelper.AssertSuccess(t, body, tt.wantOK)
		})
	}
}

func TestCreate(t *testing.T) {
	claims := models.UserClaims{UserID: 1, Email: "test@example.com", Name: "Test"}

	tests := []struct {
		name       string
		body       any
		mockFn     func(pgxmock.PgxPoolIface)
		wantStatus int
		wantOK     bool
	}{
		{
			name: "create berhasil",
			body: map[string]any{
				"name": "Monitor 4K", "price": 5000000, "stock": 3,
			},
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "description", "price", "stock",
					"category", "created_by", "created_at", "updated_at",
				}).AddRow(1, "Monitor 4K", "", 5000000.0, 3,
					"uncategorized", 1, "2024-01-01", "2024-01-01")
				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs("Monitor 4K", "", 5000000.0, 3, "uncategorized", 1).
					WillReturnRows(rows)
			},
			wantStatus: http.StatusCreated,
			wantOK:     true,
		},
		{
			name:       "name kosong",
			body:       map[string]any{"name": "", "price": 100000},
			mockFn:     func(mock pgxmock.PgxPoolIface) {},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
		{
			name:       "price negatif",
			body:       map[string]any{"name": "Produk X", "price": -1000},
			mockFn:     func(mock pgxmock.PgxPoolIface) {},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
		{
			name:       "body tidak valid",
			body:       "ini bukan json",
			mockFn:     func(mock pgxmock.PgxPoolIface) {},
			wantStatus: http.StatusBadRequest,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newProductHandler(t)
			tt.mockFn(mock)

			req := testhelper.NewAuthRequest(t, http.MethodPost, "/products", tt.body, claims)
			w := httptest.NewRecorder()

			h.CreateProduct(w, req)

			testhelper.AssertStatus(t, w.Code, tt.wantStatus)
			body := testhelper.ParseResponse(t, w)
			testhelper.AssertSuccess(t, body, tt.wantOK)
		})
	}
}

func TestDelete(t *testing.T) {
	claims := models.UserClaims{UserID: 1, Email: "test@example.com", Name: "Test"}

	tests := []struct {
		name       string
		pathID     string
		mockFn     func(pgxmock.PgxPoolIface)
		wantStatus int
	}{
		{
			name:   "delete berhasil",
			pathID: "1",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM products`).
					WithArgs(1, 1).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "bukan pemilik produk",
			pathID: "2",
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM products`).
					WithArgs(2, 1).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ID tidak valid",
			pathID:     "xyz",
			mockFn:     func(mock pgxmock.PgxPoolIface) {},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock := newProductHandler(t)
			tt.mockFn(mock)

			req := testhelper.NewAuthRequest(t, http.MethodDelete, "/products/"+tt.pathID, nil, claims)
			req.SetPathValue("id", tt.pathID)
			w := httptest.NewRecorder()

			h.DeleteProduct(w, req)

			testhelper.AssertStatus(t, w.Code, tt.wantStatus)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ekspektasi mock belum terpenuhi: %v", err)
			}
		})
	}
}
