package handlers

import (
	"log/slog"
	"net/http"

	"emeraldfox/internal/response"
	"emeraldfox/internal/service"
)

// AuthHandler expõe os endpoints públicos de cadastro e login.
type AuthHandler struct {
	base

	users *service.UserService
}

// NewAuthHandler cria o handler de autenticação.
func NewAuthHandler(users *service.UserService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{base: base{log: log}, users: users}
}

// registerRequest é o corpo aceito em POST /register.
type registerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// loginRequest é o corpo aceito em POST /login.
type loginRequest struct {
	Email string `json:"email"`
}

// Register trata POST /register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := h.decode(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	user, err := h.users.Register(r.Context(), service.RegisterInput{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	response.Created(w, user)
}

// Login trata POST /login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := h.decode(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	session, err := h.users.Login(r.Context(), service.LoginInput{Email: req.Email})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	response.OK(w, session)
}
