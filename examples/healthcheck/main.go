package main

import (
	"context"
	"log"
	"net/http"
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
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := connect.PingContext(ctx, db); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
