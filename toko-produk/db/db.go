package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=require&channel_binding=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)

	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	pgxCon, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid database url: %v", err)
	}

	pgxCon.MaxConns = 10
	pgxCon.MaxConnIdleTime = 5 * time.Minute
	pgxCon.MaxConnLifetime = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pgxCon)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("Berhasil konek ke database")
	return pool, nil
}
