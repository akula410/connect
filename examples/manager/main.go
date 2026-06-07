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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager := connect.NewManager()
	defer func() {
		if err := manager.CloseAll(); err != nil {
			log.Printf("close connections: %v", err)
		}
	}()

	master, err := manager.Open(ctx, "master", connect.Config{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
	})
	if err != nil {
		log.Fatal(err)
	}

	stats := master.Stats()
	fmt.Printf("master: open=%d idle=%d inuse=%d\n",
		stats.OpenConnections, stats.Idle, stats.InUse)

	// Retrieve a named connection anywhere in your application.
	got, ok := manager.Get("master")
	if !ok {
		log.Fatal("master connection not found")
	}
	fmt.Println("same connection pointer:", got == master)
	fmt.Println("registered names:", manager.Names())
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
