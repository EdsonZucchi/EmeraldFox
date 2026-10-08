package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"emeraldfox/internal/appctx"
	"emeraldfox/internal/apperr"
	"emeraldfox/internal/response"
)

// UserHandler expõe os endpoints protegidos de usuário.
type UserHandler struct {
	base
}

// NewUserHandler cria o handler de usuários.
func NewUserHandler(log *slog.Logger) *UserHandler {
	return &UserHandler{base: base{log: log}}
}

// Me trata GET /me devolvendo o usuário autenticado.
//
// Os dados vêm exclusivamente do contexto, preenchido pelo middleware de
// autenticação: nenhuma consulta adicional é necessária aqui.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := appctx.User(r.Context())
	if !ok {
		// Só ocorre se a rota for registrada sem o middleware de autenticação.
		h.fail(w, r, apperr.Internal(errors.New("usuário autenticado ausente no contexto")))
		return
	}

	response.OK(w, user)
}
