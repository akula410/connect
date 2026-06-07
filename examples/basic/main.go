package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/akula410/connect/v2"
)

func main() {
	user := requireEnv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	host := envOr("MYSQL_HOST", "127.0.0.1")
	port := envOr("MYSQL_PORT", "3306")
	dbName := requireEnv("MYSQL_DATABASE")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := connect.NewMySQLContext(ctx, connect.Config{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,

		MaxOpenConns:    25,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := queryVersion(ctx, db); err != nil {
		log.Fatal(err)
	}
}

func queryVersion(ctx context.Context, db *sql.DB) error {
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return err
	}
	fmt.Println("MySQL version:", version)
	return nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "error: environment variable %s is required\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
