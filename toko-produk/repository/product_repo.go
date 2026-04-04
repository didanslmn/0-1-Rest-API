package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"toko-produk/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
	DB *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{DB: db}
}

// helper untuk scan product
func scanProduct(row interface{ Scan(dest ...any) error }) (models.Product, error) {
	var p models.Product
	var description sql.NullString
	err := row.Scan(&p.ID, &p.Name, &description, &p.Price, &p.Stock, &p.Category, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return models.Product{}, err
	}
	if description.Valid {
		p.Description = description.String
	}
	return p, nil
}

func (r *ProductRepository) GetAll(ctx context.Context, filter models.ProdyctFilter) ([]models.Product, error) {
	query := "SELECT id, name, description, price, stock, category, created_by, created_at, updated_at FROM products WHERE 1=1"
	args := []any{}
	argIndex := 1

	if filter.Category != "" {
		query += fmt.Sprintf(" AND LOWER(category) = $%d", argIndex)
		args = append(args, strings.ToLower(filter.Category))
		argIndex++
	}

	if filter.MinPrice != "" {
		query += fmt.Sprintf(" AND price >= $%d::numeric", argIndex)
		args = append(args, filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice != "" {
		query += fmt.Sprintf(" AND price <= $%d::numeric", argIndex)
		args = append(args, filter.MaxPrice)
		argIndex++
	}

	query += " ORDER BY id"

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id int) (models.Product, error) {
	query := "SELECT id, name, description, price, stock, category, created_by, created_at, updated_at FROM products WHERE id = $1"
	row := r.DB.QueryRow(ctx, query, id)
	return scanProduct(row)
}

func (r *ProductRepository) Create(ctx context.Context, req models.CreateProductRequest, UserID int) (models.Product, error) {
	category := req.Category
	if category == "" {
		category = "uncategorized"
	}
	query := `
		INSERT INTO products (name, description, price, stock, category, created_by) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, name, description, price, stock, category, created_by, created_at, updated_at
	`
	row := r.DB.QueryRow(ctx, query, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.Price, req.Stock, category, UserID)
	return scanProduct(row)
}

func (r *ProductRepository) Update(ctx context.Context, req models.UpdateProductRequest, id int, UserID int) (models.Product, error) {
	setClause := []string{"updated_at = NOW()"}
	args := []any{}
	argIndex := 1

	if req.Name != nil {
		setClause = append(setClause, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, strings.TrimSpace(*req.Name))
		argIndex++
	}

	if req.Description != nil {
		setClause = append(setClause, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, strings.TrimSpace(*req.Description))
		argIndex++
	}

	if req.Price != nil {
		setClause = append(setClause, fmt.Sprintf("price = $%d::numeric", argIndex))
		args = append(args, *req.Price)
		argIndex++
	}

	if req.Stock != nil {
		setClause = append(setClause, fmt.Sprintf("stock = $%d", argIndex))
		args = append(args, *req.Stock)
		argIndex++
	}

	if req.Category != nil {
		setClause = append(setClause, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, strings.TrimSpace(*req.Category))
		argIndex++
	}
	// hanya pemilik produk yang bisa update
	args = append(args, id, UserID)
	query := fmt.Sprintf(`
		UPDATE products SET %s WHERE id = $%d AND created_by = $%d 
		RETURNING id, name, description, price, stock, category, created_by, created_at, updated_at
	`, strings.Join(setClause, ", "), argIndex, argIndex+1)

	row := r.DB.QueryRow(ctx, query, args...)
	return scanProduct(row)
}

func (r *ProductRepository) Delete(ctx context.Context, id int, UserID int) error {
	query := "DELETE FROM products WHERE id = $1 AND created_by = $2"
	result, err := r.DB.Exec(ctx, query, id, UserID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return &NotFoundError{"produk tidak ditemukan atau kamu bukan pemiliknya"}
	}
	return nil
}

type NotFoundError struct{ Msg string }
type ValidationError struct{ Msg string }

func (e *NotFoundError) Error() string   { return e.Msg }
func (e *ValidationError) Error() string { return e.Msg }
