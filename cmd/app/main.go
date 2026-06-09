package main

import (
	"context"
	"log"
	"net/http"

	"github.com/NIDJEL/MiniBank/internal/server"
	storage "github.com/NIDJEL/MiniBank/internal/storeage"
	"github.com/redis/go-redis/v9"
)

func main() {
	databaseURL := "postgres://minibank:minibankpass@localhost:5432/minibank?sslmode=disable"

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	db, err := storage.OpenPostgres(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	srv, err := server.New(db, rdb)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("MiniBank started on :8080")

	err = http.ListenAndServe(":8080", srv.Handler())
	if err != nil {
		log.Fatal(err)
	}
}
