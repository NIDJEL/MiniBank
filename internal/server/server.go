package server

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	mux       *http.ServeMux
	templates *template.Template
	db        *sql.DB
}

func New(db *sql.DB) (*Server, error) {

	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:       http.NewServeMux(),
		templates: tmpl,
		db:        db,
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

	s.mux.HandleFunc("GET /ready", s.readyHandler)
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

	if username == "" || email == "" || password == "" {
		http.Error(w, "заполните все поля", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "cannot hash password", http.StatusInternalServerError)
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var userID int64

	err = tx.QueryRow(
		`INSERT INTO users (username, email, password_hash)
			   VALUES ($1, $2, $3)
			   RETURNING id`,
		username,
		email,
		string(passwordHash),
	).Scan(&userID)
	if err != nil {
		http.Error(w, "cannot create user", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO accounts (user_id, balance)
			   values ($1, $2)`,
		userID,
		0,
	)

	if err != nil {
		http.Error(w, "cannot create account", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "cannot commit transaction", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("User created. ID: " + fmt.Sprint(userID)))
}

func (s *Server) showLoginHandler(w http.ResponseWriter, r *http.Request) {
	s.render(w, "login.html")
}

func (s *Server) submitLoginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Fild textboxes", http.StatusBadRequest)
		return
	}

	var userID int
	var username string
	var passwordHash string

	err := s.db.QueryRow(`SELECT id, username, password_hash
		FROM users
		WHERE email = $1`,
		email,
	).Scan(&userID, &username, &passwordHash)

	if err != nil {
		http.Error(w, "incorrect password or login\n", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		http.Error(w, "incorrect password or login\n", http.StatusUnauthorized)
		return
	}

	w.Write([]byte("Successful login. Hi, " + username))
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.db.Ping()
	if err != nil {
		http.Error(w, "database is not ready", http.StatusServiceUnavailable)
		return
	}

	w.Write([]byte("database is ready\n"))
}
