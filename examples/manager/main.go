package main

import (
	"context"
	"log"
	"time"

	"github.com/akula410/connect/v2"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager := connect.NewManager()
	defer func() {
		if err := manager.CloseAll(); err != nil {
			log.Printf("close connections: %v", err)
		}
	}()

	master, err := manager.Open(ctx, "master", connect.Config{
		User:     "app_user",
		Password: "app_password",
		Host:     "127.0.0.1",
		Port:     "3306",
		DBName:   "app_db",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("master connection is ready", master.Stats())

	sameMaster, ok := manager.Get("master")
	if !ok {
		log.Fatal("master connection not found")
	}

	log.Println("same connection:", sameMaster == master)
}
