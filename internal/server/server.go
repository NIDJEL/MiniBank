package server

import (
	"fmt"
	"html/template"
	"net/http"
)

type Server struct {
	mux       *http.ServeMux
	templates *template.Template
}

func New() (*Server, error) {

	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:       http.NewServeMux(),
		templates: tmpl,
	}

	s.routes()

	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.homeHandler)
	s.mux.HandleFunc("GET /health", s.healthHandler)

	s.mux.HandleFunc("GET /register", s.showRegisterHandler)
	s.mux.HandleFunc("POST /register", s.submitRegisterHandler)

	s.mux.HandleFunc("GET /login", s.showLoginHandler)
	s.mux.HandleFunc("POST /login", s.submitLoginHandler)
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

func (s *Server) render(w http.ResponseWriter, name string) {
	err := s.templates.ExecuteTemplate(w, name, nil)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) showRegisterHandler(w http.ResponseWriter, r *http.Request) {
	s.render(w, "register.html")
}

func (s *Server) submitRegisterHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	w.Write([]byte("New user: " + username + "/" + email + "/" + password))
}

func (s *Server) showLoginHandler(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login.html")
}

func (s *Server) submitLoginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	w.Write([]byte("Get login: " + email + "/" + password))

}
