package main

import (
	"log"
	"net/http"

	"github.com/NIDJEL/MiniBank/internal/server"
)

func main() {
	srv := server.New()

	log.Println("MiniBank started on :8080")

	err := http.ListenAndServe(":8080", srv.Handler())
	if err != nil {
		log.Fatal(err)
	}
}
