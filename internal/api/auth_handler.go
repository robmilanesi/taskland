package api

import (
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/robmilanesi/taskland/internal/auth"
	"github.com/robmilanesi/taskland/internal/httpx"
	"github.com/robmilanesi/taskland/internal/models"
	"github.com/robmilanesi/taskland/internal/repository"
)

// AuthHandler serves the registration and login endpoints.
type AuthHandler struct {
	users  repository.UserRepository
	issuer *auth.Issuer
}

// NewAuthHandler returns an AuthHandler backed by the given user repository and
// token issuer.
func NewAuthHandler(users repository.UserRepository, issuer *auth.Issuer) *AuthHandler {
	return &AuthHandler{users: users, issuer: issuer}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := req.validate(); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	user, err := h.users.CreateUser(r.Context(), models.User{
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: string(hash),
	})
	if errors.Is(err, repository.ErrEmailTaken) {
		httpx.WriteError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, userResponse{ID: user.ID, Email: user.Email})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}

	user, err := h.users.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if errors.Is(err, repository.ErrUserNotFound) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.issuer.Issue(user.ID)
	if err != nil {
		httpx.WriteISE(w)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokenResponse{Token: token})
}
