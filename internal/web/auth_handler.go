package web

import (
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
	"github.com/robmilanesi/taskland/internal/web/views"
)

// AuthHandler serves the login/register/logout pages and form submissions.
type AuthHandler struct {
	store  *repository.Store
	issuer *auth.Issuer
}

// NewAuthHandler returns an AuthHandler backed by store and issuer.
func NewAuthHandler(store *repository.Store, issuer *auth.Issuer) *AuthHandler {
	return &AuthHandler{store: store, issuer: issuer}
}

// LoginForm handles GET /login.
func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, views.Login(""))
}

// Login handles POST /login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render(w, r, views.Login("invalid form submission"))
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := h.store.Users.GetUserByEmail(r.Context(), email)
	if errors.Is(err, repository.ErrUserNotFound) {
		render(w, r, views.Login("invalid credentials"))
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		render(w, r, views.Login("invalid credentials"))
		return
	}

	if err := setSession(w, r, h.issuer, user.ID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RegisterForm handles GET /register.
func (h *AuthHandler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, views.Register(""))
}

// Register handles POST /register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render(w, r, views.Register("invalid form submission"))
		return
	}
	form := registerForm{
		Email:    strings.TrimSpace(r.FormValue("email")),
		Password: r.FormValue("password"),
	}
	if err := form.validate(); err != nil {
		render(w, r, views.Register(err.Error()))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.store.RegisterAccount(r.Context(), models.User{
		Email:        form.Email,
		PasswordHash: string(hash),
	})
	if errors.Is(err, repository.ErrEmailTaken) {
		render(w, r, views.Register("email already registered"))
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := setSession(w, r, h.issuer, user.ID); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles POST /logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
