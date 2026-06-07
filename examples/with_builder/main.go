//go:build ignore

// Example: connect + github.com/akula410/builder
//
// This file shows integration with the builder package using its modern API
// (NewExecutor, Select, Eq, QueryContext). It requires a newer version of
// github.com/akula410/builder that provides these functions.
//
// Run against a real MySQL server:
//
//	CONNECT_MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/app_db' go run examples/with_builder/main.go
package main

import (
	"context"
	"log"
	"time"

	sqlbuilder "github.com/akula410/builder"
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
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// connect returns *sql.DB — pass it directly to the builder executor.
	exec := sqlbuilder.NewExecutor(db)

	rows, err := exec.QueryContext(ctx,
		sqlbuilder.Select("id", "name", "email").
			From("users").
			Where(sqlbuilder.Eq("status", "active")).
			OrderBy("id", sqlbuilder.Desc).
			Limit(10),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			log.Fatal(err)
		}
		log.Printf("user id=%d name=%s email=%s", id, name, email)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
