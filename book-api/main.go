package main

import (
	"book-api/db"
	"book-api/handler"
	"book-api/repository"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	// load .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("file .env tidak ditemukan")
	}

	// connect to database
	db, err := db.Connect()
	if err != nil {
		log.Fatal("gagal konek ke database: ", err)
	}
	defer db.Close()

	// Inisialisasi dependency
	bookRepo := repository.NewBookRepository(db)
	bookHandler := handler.NewBookHandler(bookRepo)

	// Register routes
	http.HandleFunc("/books", bookHandler.HandleBooks)
	http.HandleFunc("/books/", bookHandler.HandleBookByID)

	port := ":8080"
	log.Printf("Book API berjalan di http://localhost%s", port)
	log.Printf("Endpoints:")
	log.Printf("  GET    /books               → semua buku (filter: ?author=&genre=)")
	log.Printf("  POST   /books               → tambah buku")
	log.Printf("  GET    /books/{id}          → detail buku")
	log.Printf("  PUT    /books/{id}          → update buku")
	log.Printf("  DELETE /books/{id}          → hapus buku")
	log.Printf("  GET    /books/stats         → statistik koleksi")
	log.Fatal(http.ListenAndServe(port, nil))
}
