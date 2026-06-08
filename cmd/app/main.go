package main

import (
	"log"
	"net/http"

	"github.com/NIDJEL/MiniBank/internal/server"
	storage "github.com/NIDJEL/MiniBank/internal/storeage"
)

func main() {
	databaseURL := "postgres://minibank:minibankpass@localhost:5432/minibank?sslmode=disable"

	db, err := storage.OpenPostgres(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	srv, err := server.New(db)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("MiniBank started on :8080")

	err = http.ListenAndServe(":8080", srv.Handler())
	if err != nil {
		log.Fatal(err)
	}
}
