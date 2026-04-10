package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"toko-produk/db"
	"toko-produk/handler"
	Middleware "toko-produk/middleware"
	"toko-produk/repository"
)

func main() {
	// load .env
	// if err := godotenv.Load(); err != nil {
	// 	fmt.Println("Gagal load .env", err)
	// 	return
	// }
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

	// Auth routes (public)
	mux.HandleFunc("POST /register", userHandler.Register)
	mux.HandleFunc("POST /login", userHandler.Login)

	// product routes (public)
	mux.HandleFunc("GET /products", productHandler.GetAllProducts)
	mux.HandleFunc("GET /product/{id}", productHandler.GetProductByID)

	// product routes (private)
	mux.Handle("GET /product/me", Middleware.Auth(productHandler.GetMyProduct))
	mux.Handle("POST /product", Middleware.Auth(productHandler.CreateProduct))
	mux.Handle("PUT /product/{id}", Middleware.Auth(productHandler.UpdateProduct))
	mux.Handle("DELETE /product/{id}", Middleware.Auth(productHandler.DeleteProduct))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		// Cek koneksi database juga
		if err := pool.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"reason": "database tidak merespons",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	})

	loggedMux := Middleware.Logger(mux)
	fmt.Println("Server berjalan di port 8080")
	if err := http.ListenAndServe(":8080", loggedMux); err != nil {
		fmt.Println("Gagal menjalankan server", err)
	}
}
