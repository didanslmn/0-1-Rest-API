package models

import "time"

type Book struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Author    string     `json:"author"`
	Genre     string     `json:"genre"`
	Price     float64    `json:"price"`
	Stock     int        `json:"stock"`
	Published *time.Time `json:"published"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreateBookRequest struct {
	Title     string  `json:"title" binding:"required"`
	Author    string  `json:"author" binding:"required"`
	Genre     string  `json:"genre" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
	Stock     int     `json:"stock" binding:"required"`
	Published string  `json:"published" binding:"required"`
}

type UpdateBookRequest struct {
	ID        *int     `json:"id"`
	Title     *string  `json:"title"`
	Author    *string  `json:"author"`
	Genre     *string  `json:"genre"`
	Price     *float64 `json:"price"`
	Stock     *int     `json:"stock"`
	Published *string  `json:"published"`
}

type BookFilter struct {
	Author string
	Genre  string
}
