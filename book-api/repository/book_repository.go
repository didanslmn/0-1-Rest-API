package repository

import (
	"book-api/models"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BookRepository struct {
	db *pgxpool.Pool
}

// NewBookRepository membuat instance baru dari BookRepository
func NewBookRepository(db *pgxpool.Pool) *BookRepository {
	return &BookRepository{db: db}
}

// scanBook membantu membaca satu row hasil query ke struct Book
func scanBook(row interface{ Scan(dest ...any) error }) (models.Book, error) {
	var b models.Book
	err := row.Scan(
		&b.ID,
		&b.Title,
		&b.Author,
		&b.Genre,
		&b.Price,
		&b.Stock,
		&b.Published,
		&b.CreatedAt,
	)
	return b, err
}

// GetAll mengembalikan semua buku dengan filter opsional
func (r *BookRepository) GetAll(filter models.BookFilter) ([]models.Book, error) {
	query := "SELECT id,title,author,genre,price,stock,published,created_at FROM books WHERE 1 = 1"
	args := []any{}
	argsIdx := 1

	if filter.Author != "" {
		query += fmt.Sprintf(" AND author ILIKE $%d", argsIdx)
		args = append(args, "%"+filter.Author+"%")
		argsIdx++
	}

	if filter.Genre != "" {
		query += fmt.Sprintf(" AND genre = $%d", argsIdx)
		args = append(args, filter.Genre)
		argsIdx++
	}

	query += " ORDER BY id"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// inisialisasi slice kosong untuk menampung hasil
	books := []models.Book{}
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, rows.Err()
}

func (r *BookRepository) GetByID(id int) (models.Book, error) {
	query := "SELECT id,title,author,genre,price,stock,published,created_at FROM books WHERE id = $1"
	row := r.db.QueryRow(context.Background(), query, id)
	return scanBook(row)
}

func (r *BookRepository) Create(req models.CreateBookRequest) (models.Book, error) {
	query := "INSERT INTO books (title,author,genre,price,stock,published) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id,title,author,genre,price,stock,published,created_at"

	// parse publised bila diisi
	var published *time.Time
	if req.Published != "" {
		t, err := time.Parse("2006-01-02", req.Published)
		if err != nil {
			return models.Book{}, &ValidationError{"format published salah, gunakan: YYYY-MM-DD"}
		}
		published = &t
	}
	row := r.db.QueryRow(context.Background(), query, strings.TrimSpace(req.Title), strings.TrimSpace(req.Author), strings.TrimSpace(req.Genre), req.Price, req.Stock, published)
	return scanBook(row)
}

func (r *BookRepository) Update(id int, req models.UpdateBookRequest) (models.Book, error) {
	// bangun query update secara dinamis	- hanya update filed yang dikirim
	setClause := []string{"updated_at=NOW()"}
	args := []any{}
	argsIndex := 1

	if req.Title != nil {
		setClause = append(setClause, "title=$"+strconv.Itoa(argsIndex))
		args = append(args, strings.TrimSpace(*req.Title))
		argsIndex++
	}
	if req.Author != nil {
		setClause = append(setClause, "author=$"+strconv.Itoa(argsIndex))
		argsIndex++
	}
	if req.Genre != nil {
		setClause = append(setClause, "genre=$"+strconv.Itoa(argsIndex))
		argsIndex++
	}
	if req.Price != nil {
		setClause = append(setClause, "price=$"+strconv.Itoa(argsIndex))
		argsIndex++
	}
	if req.Stock != nil {
		setClause = append(setClause, "stock=$"+strconv.Itoa(argsIndex))
		argsIndex++
	}
	if req.Published != nil {
		if *req.Published == "" {
			setClause = append(setClause, "published=$"+strconv.Itoa(argsIndex))
			args = append(args, nil)
			argsIndex++
		} else {
			t, err := time.Parse("2006-01-02", *req.Published)
			if err != nil {
				return models.Book{}, &ValidationError{"format published salah, gunakan: YYYY-MM-DD"}
			}
			setClause = append(setClause, "published=$"+strconv.Itoa(argsIndex))
			args = append(args, t)
			argsIndex++
		}
	}
	args = append(args, id)
	query := "UPDATE books SET " + strings.Join(setClause, ",") + "WHERE id =$" + strconv.Itoa(argsIndex) + "RETURNING id,title,author,genre,price,stock,published,created_at, updated_at"

	row := r.db.QueryRow(context.Background(), query, args...)
	b, err := scanBook(row)
	if err == sql.ErrNoRows {
		return models.Book{}, &NotFoundError{"buku tidak ditemukan"}
	}
	if err != nil {
		return models.Book{}, err
	}
	return b, nil
}

func (r *BookRepository) Delete(id int) error {
	result, err := r.db.Exec(context.Background(), "DELETE FROM books WHERE id = $1", id)
	if err != nil {
		return err
	}

	rows := result.RowsAffected()
	if rows == 0 {
		return &NotFoundError{"buku tidak ditemukan"}
	}
	return nil
}

func (r *BookRepository) GetStats() (map[string]any, error) {
	query := `
	SELECT
	COUNT(*),
	COALESCE(AVG(price),0) as avg_price,
	COALESCE(SUM(stock),0) as total_stock,
	COALESCE(MIN(price),0) as min_price,
	COALESCE(MAX(price),0) as max_price,
	COUNT(DISTINCT author) as total_author,
	COUNT(DISTINCT genre) as total_genre
	FROM books
	`
	row := r.db.QueryRow(context.Background(), query)
	var total, avgPrice, totalStock, minPrice, maxPrice, totalAuthor, totalGenre int
	err := row.Scan(&total, &avgPrice, &totalStock, &minPrice, &maxPrice, &totalAuthor, &totalGenre)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_buku":      total,
		"rata_rata_harga": avgPrice,
		"total_stok":      totalStock,
		"harga_termurah":  minPrice,
		"harga_termahal":  maxPrice,
		"total_penulis":   totalAuthor,
		"total_genre":     totalGenre,
	}, nil
}

// custom error

type ValidationError struct {
	Msg string
}
type NotFoundError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}
func (e *NotFoundError) Error() string {
	return e.Msg
}
