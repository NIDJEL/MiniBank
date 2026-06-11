package server

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	mux       *http.ServeMux
	templates *template.Template
	db        *sql.DB
	redis     *redis.Client
}

type dashboardData struct {
	Username string
	Email    string
	Balance  string
	Currency string
}

func New(db *sql.DB, rdb *redis.Client) (*Server, error) {

	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:       http.NewServeMux(),
		templates: tmpl,
		db:        db,
		redis:     rdb,
	}

	s.routes()

	return s, nil
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	s.mux.HandleFunc("GET /", s.homeHandler)
	s.mux.HandleFunc("GET /health", s.healthHandler)
	s.mux.HandleFunc("GET /register", s.showRegisterHandler)
	s.mux.HandleFunc("POST /register", s.submitRegisterHandler)
	s.mux.HandleFunc("GET /login", s.showLoginHandler)
	s.mux.HandleFunc("POST /login", s.submitLoginHandler)
	s.mux.HandleFunc("GET /ready", s.readyHandler)
	s.mux.HandleFunc("GET /redis-ping", s.redisPingHandler)
	s.mux.HandleFunc("GET /dashboard", s.dashboardHandler)
	s.mux.HandleFunc("GET /logout", s.logoutHandler)
	s.mux.HandleFunc("GET /loguot", s.logoutHandler)
	s.mux.HandleFunc("GET /deposit", s.depositPageHandler)
	s.mux.HandleFunc("POST /deposit", s.submitDepositHandler)
	s.mux.HandleFunc("GET /withdraw", s.withdrawPageHandler)
	s.mux.HandleFunc("POST /withdraw", s.submitWithdrawHandler)
}

func (s *Server) depositPageHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := s.currentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err := s.templates.ExecuteTemplate(w, "deposit.html", nil)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) submitDepositHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.currentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	amount := r.FormValue("amount")

	value, err := strconv.ParseFloat(amount, 64)
	if err != nil || value <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec(`
		UPDATE accounts
		SET balance = balance + $1::numeric
		WHERE user_id = $2`, amount, userID)
	if err != nil {
		http.Error(w, "cannot deposit memory", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) withdrawPageHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := s.currentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	err := s.templates.ExecuteTemplate(w, "withdraw.html", nil)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) submitWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.currentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	amount := r.FormValue("amount")

	value, err := strconv.ParseFloat(amount, 64)
	if err != nil || value <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return
	}

	result, err := s.db.Exec(`
		UPDATE accounts
		SET balance = balance - $1::numeric
		WHERE user_id = $2 AND balance >= $1::numeric
	`, amount, userID)
	if err != nil {
		http.Error(w, "cannot withdraw money", http.StatusInternalServerError)
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "cannot check withdraw result", http.StatusInternalServerError)
		return
	}

	if rows == 0 {
		http.Error(w, "not enough money", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	s.render(w, "home.html")
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

	http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	sessionID, err := newSessionId()
	if err != nil {
		http.Error(w, "cannot create session", http.StatusInternalServerError)
		return
	}

	err = s.redis.Set(
		r.Context(),
		"session:"+sessionID,
		userID,
		24*time.Hour).Err()

	if err != nil {
		http.Error(w, "cannot save session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) currentUserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, false
	}

	userIDText, err := s.redis.Get(r.Context(), "session:"+cookie.Value).Result()
	if err != nil {
		return 0, false
	}

	userID, err := strconv.Atoi(userIDText)
	if err != nil {
		return 0, false
	}

	return userID, true
}

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.currentUserID(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var data dashboardData

	err := s.db.QueryRow(`
		SELECT u.username, u.email, a.balance::text, a.currency
		FROM users u
		JOIN accounts a ON a.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(
		&data.Username,
		&data.Email,
		&data.Balance,
		&data.Currency,
	)
	if err != nil {
		http.Error(w, "cannot load dashboard", http.StatusInternalServerError)
		return
	}

	err = s.templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, "temlate error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		_ = s.redis.Del(r.Context(), "session:"+cookie.Value).Err()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	err := s.db.Ping()
	if err != nil {
		http.Error(w, "database is not ready", http.StatusServiceUnavailable)
		return
	}

	w.Write([]byte("database is ready\n"))
}

func (s *Server) redisPingHandler(w http.ResponseWriter, r *http.Request) {
	err := s.redis.Ping(r.Context()).Err()
	if err != nil {
		http.Error(w, "redis is not reday", http.StatusServiceUnavailable)
		return
	}

	w.Write([]byte("redis is ready"))
}

func newSessionId() (string, error) {
	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
