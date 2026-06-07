// Example: observability via sql.DBStats.
//
// Run:
//
//	MYSQL_USER=app MYSQL_PASSWORD=secret MYSQL_DATABASE=mydb go run examples/stats/main.go
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

	// --- Single connection stats ---

	db, err := connect.NewMySQLContext(ctx, connect.Config{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// *sql.DB exposes Stats() directly — no wrapper needed.
	stats := db.Stats()
	fmt.Println("=== single connection ===")
	fmt.Printf("OpenConnections:  %d\n", stats.OpenConnections)
	fmt.Printf("InUse:            %d\n", stats.InUse)
	fmt.Printf("Idle:             %d\n", stats.Idle)
	fmt.Printf("WaitCount:        %d\n", stats.WaitCount)
	fmt.Printf("WaitDuration:     %v\n", stats.WaitDuration)
	fmt.Printf("MaxIdleClosed:    %d\n", stats.MaxIdleClosed)
	fmt.Printf("MaxLifetimeClosed:%d\n", stats.MaxLifetimeClosed)

	// --- Manager stats ---

	manager := connect.NewManager()
	defer manager.CloseAll()

	for _, name := range []string{"master", "replica"} {
		_, err := manager.Open(ctx, name, connect.Config{
			User:     user,
			Password: password,
			Host:     host,
			Port:     port,
			DBName:   dbName,
		})
		if err != nil {
			log.Fatalf("open %s: %v", name, err)
		}
	}

	fmt.Println("\n=== manager.Stats(name) ===")
	if s, ok := manager.Stats("master"); ok {
		fmt.Printf("master: open=%d inuse=%d idle=%d\n", s.OpenConnections, s.InUse, s.Idle)
	}

	fmt.Println("\n=== manager.StatsAll() ===")
	for name, s := range manager.StatsAll() {
		fmt.Printf("%s: open=%d inuse=%d idle=%d waitCount=%d\n",
			name, s.OpenConnections, s.InUse, s.Idle, s.WaitCount)
	}
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
