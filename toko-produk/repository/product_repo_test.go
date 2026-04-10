package repository_test

import (
	"context"
	"testing"
	"toko-produk/models"
	"toko-produk/repository"

	"github.com/pashagolub/pgxmock/v3"
)

// newMockRepo membuat mock repository untuk keperluan test denagn database mock
func newMockRepo(t *testing.T) (*repository.ProductRepository, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("gagal membuat mock database: %v", err)
	}
	t.Cleanup(func() { mock.Close() })
	return repository.NewProductRepository(mock), mock
}
func TestGetByID(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		mockFn  func(pgxmock.PgxPoolIface)
		want    models.Product
		wantErr bool
	}{
		{
			name: "produk ditemukan",
			id:   1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "description", "price", "stock",
					"category", "created_by", "created_at", "updated_at",
				}).AddRow(1, "Laptop Gaming", "RTX 4060", 15000000.0, 5,
					"elektronik", 1, "2024-01-01", "2024-01-01")

				mock.ExpectQuery(`SELECT id, name, description`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want: models.Product{
				ID:       1,
				Name:     "Laptop Gaming",
				Price:    15000000,
				Stock:    5,
				Category: "elektronik",
			},
			wantErr: false,
		},
		{
			name: "produk tidak ditemukan",
			id:   999,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, name, description`).
					WithArgs(999).
					WillReturnRows(pgxmock.NewRows(nil))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			tt.mockFn(mock)

			got, err := repo.GetByID(context.Background(), tt.id)

			if tt.wantErr {
				if err == nil {
					t.Error("harusnya error, tapi tidak ada")
				}
				return
			}
			if err != nil {
				t.Fatalf("error tidak terduga: %v", err)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID: got %d, want %d", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name: got %q, want %q", got.Name, tt.want.Name)
			}

			// Pastikan semua ekspektasi mock terpenuhi
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ekspektasi mock belum terpenuhi: %v", err)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.CreateProductRequest
		userID  int
		mockFn  func(pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "create berhasil",
			req: models.CreateProductRequest{
				Name:  "Keyboard Mechanical",
				Price: 750000,
				Stock: 20,
			},
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "description", "price", "stock",
					"category", "created_by", "created_at", "updated_at",
				}).AddRow(1, "Keyboard Mechanical", nil, 750000.0, 20,
					"uncategorized", 1, "2024-01-01", "2024-01-01")

				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs("Keyboard Mechanical", "", 750000.0, 20, "uncategorized", 1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "category default ke uncategorized",
			req: models.CreateProductRequest{
				Name:  "Mouse Gaming",
				Price: 350000,
				Stock: 15,
				// Category sengaja dikosongkan
			},
			userID: 1,
			mockFn: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "description", "price", "stock",
					"category", "created_by", "created_at", "updated_at",
				}).AddRow(2, "Mouse Gaming", nil, 350000.0, 15,
					"uncategorized", 1, "2024-01-01", "2024-01-01")

				mock.ExpectQuery(`INSERT INTO products`).
					WithArgs("Mouse Gaming", "", 350000.0, 15, "uncategorized", 1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)
			tt.mockFn(mock)

			got, err := repo.Create(context.Background(), tt.req, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Error("harusnya error, tapi tidak ada")
				}
				return
			}
			if err != nil {
				t.Fatalf("error tidak terduga: %v", err)
			}
			if got.Name != tt.req.Name {
				t.Errorf("Name: got %q, want %q", got.Name, tt.req.Name)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ekspektasi mock belum terpenuhi: %v", err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name         string
		id, userID   int
		rowsAffected int64
		wantErr      bool
		wantNotFound bool
	}{
		{
			name:         "delete berhasil",
			id:           1,
			userID:       1,
			rowsAffected: 1,
			wantErr:      false,
		},
		{
			name:         "produk tidak ditemukan atau bukan pemilik",
			id:           99,
			userID:       1,
			rowsAffected: 0,
			wantNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)

			mock.ExpectExec(`DELETE FROM products`).
				WithArgs(tt.id, tt.userID).
				WillReturnResult(pgxmock.NewResult("DELETE", tt.rowsAffected))

			err := repo.Delete(context.Background(), tt.id, tt.userID)

			if tt.wantNotFound {
				if _, ok := err.(*repository.NotFoundError); !ok {
					t.Errorf("harusnya NotFoundError, dapat: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("error tidak terduga: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ekspektasi mock belum terpenuhi: %v", err)
			}
		})
	}
}
