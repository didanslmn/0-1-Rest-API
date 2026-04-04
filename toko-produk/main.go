package main

import (
	"fmt"
	"net/http"
	"toko-produk/db"
	"toko-produk/handler"
	Middleware "toko-produk/middleware"
	"toko-produk/repository"

	"github.com/joho/godotenv"
)

func main() {
	// load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Gagal load .env", err)
		return
	}
	// panggil function connect
	pool, err := db.Connect()
	if err != nil {
		fmt.Println("Gagal konek ke database", err)
		return
	}

	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	userHandler := handler.NewAuthHandler(userRepo)

	productRepo := repository.NewProductRepository(pool)
	productHandler := handler.NewProductHandler(productRepo)

	// router
	mux := http.NewServeMux()

	// user routes (public)
	mux.HandleFunc("POST /register", userHandler.Register)
	mux.HandleFunc("POST /login", userHandler.Login)

	// product routes (public)
	mux.HandleFunc("GET /products", productHandler.GetAllProducts)
	mux.HandleFunc("GET /product/{id}", productHandler.GetProductByID)

	// product routes (private)
	mux.Handle("POST /product", Middleware.Auth(productHandler.CreateProduct))
	mux.Handle("PUT /product/{id}", Middleware.Auth(productHandler.UpdateProduct))
	mux.Handle("DELETE /product/{id}", Middleware.Auth(productHandler.DeleteProduct))

	fmt.Println("Server berjalan di port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Gagal menjalankan server", err)
	}
}
