// Example: startup retry with exponential backoff.
//
// Useful when the application starts before MySQL is ready (e.g., Docker Compose).
// This is a one-time startup retry — not a runtime reconnection mechanism.
//
// Run:
//
//	MYSQL_USER=app MYSQL_PASSWORD=secret MYSQL_DATABASE=mydb go run examples/retry/main.go
package main

import (
	"context"
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

	cfg := connect.Config{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
	}

	retry := connect.RetryConfig{
		Attempts: 5,
		MinDelay: 500 * time.Millisecond,
		MaxDelay: 5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("connecting to MySQL (up to %d attempts)...", retry.Attempts)

	db, err := connect.NewMySQLContextWithRetry(ctx, cfg, retry)
	if err != nil {
		log.Fatalf("could not connect after %d attempts: %v", retry.Attempts, err)
	}
	defer db.Close()

	log.Println("connected successfully")

	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		log.Fatal(err)
	}
	fmt.Println("MySQL version:", version)
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
