package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/akula410/connect/v2"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := connect.NewMySQLContext(ctx, connect.Config{
		User:     "app_user",
		Password: "app_password",
		Host:     "127.0.0.1",
		Port:     "3306",
		DBName:   "app_db",

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
