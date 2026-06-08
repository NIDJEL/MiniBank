package main

import (
	"log"
	"net/http"

	"github.com/NIDJEL/MiniBank/internal/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("MiniBank started on :8080")

	err = http.ListenAndServe(":8080", srv.Handler())
	if err != nil {
		log.Fatal(err)
	}
}
