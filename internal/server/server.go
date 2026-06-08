package server

import (
	"fmt"
	"net/http"
)

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}

	s.routes()

	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.homeHandler)
	s.mux.HandleFunc("GET /health", s.healthHandler)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "MiniBank is running\n")
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK\n")
}
