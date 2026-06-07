// Example: connection pool sizing for high-load applications.
//
// Formula used here:
//
//	MySQL max_connections     = 300
//	Reserved (admin/system)  = 30
//	Application instances    = 6
//	Available per instance   = (300 - 30) / 6 = 45
//
// Run:
//
//	MYSQL_USER=app MYSQL_PASSWORD=secret MYSQL_DATABASE=mydb go run examples/highload_config/main.go
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

	cfg := connect.DefaultConfig()
	cfg.User = user
	cfg.Password = password
	cfg.Host = host
	cfg.Port = port
	cfg.DBName = dbName

	// Tune these values for your specific setup.
	// See README: "Connection pool sizing for high-load applications".
	cfg.MaxOpenConns = 40
	cfg.MaxIdleConns = 20
	cfg.ConnMaxLifetime = 5 * time.Minute // keep below MySQL wait_timeout (default 8h)
	cfg.ConnMaxIdleTime = 3 * time.Minute

	// Tight timeouts prevent cascading failures under load.
	cfg.Timeout = 3 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := connect.NewMySQLContext(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stats := db.Stats()
	fmt.Printf("connected: open=%d idle=%d inuse=%d maxOpen=%d\n",
		stats.OpenConnections, stats.Idle, stats.InUse, stats.MaxOpenConnections)
	fmt.Println("pool is ready for high-load traffic")
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
