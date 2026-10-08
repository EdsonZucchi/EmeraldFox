// Package routes declara as rotas da API e a composição dos middlewares.
package routes

import (
	"log/slog"
	"net/http"

	"emeraldfox/internal/auth"
	"emeraldfox/internal/config"
	"emeraldfox/internal/handlers"
	"emeraldfox/internal/middleware"
	"emeraldfox/internal/response"
	"emeraldfox/internal/service"

	"github.com/gorilla/mux"
)

// Dependencies reúne tudo o que o roteador precisa receber por injeção.
//
// Novos módulos financeiros passam a expor seus handlers aqui, sem alterar a
// composição dos middlewares.
type Dependencies struct {
	Config *config.Config
	Logger *slog.Logger
	Tokens *auth.TokenManager
	Users  *service.UserService
	Auth   *handlers.AuthHandler
	User   *handlers.UserHandler
	Health *handlers.HealthHandler
}

// New monta o roteador da aplicação já envolvido pelos middlewares globais.
//
// Ordem dos middlewares globais (do mais externo para o mais interno):
//
//	RequestID -> Logging -> Recovery -> CORS
//
// Assim toda requisição — inclusive 404 e 405 — recebe identificador, é
// registrada no log com o status final e nunca derruba o servidor por panic.
func New(deps Dependencies) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)

	router.NotFoundHandler = http.HandlerFunc(notFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowed)

	// Rotas públicas.
	router.HandleFunc("/health", deps.Health.Health).Methods(http.MethodGet)
	router.HandleFunc("/register", deps.Auth.Register).Methods(http.MethodPost)
	router.HandleFunc("/login", deps.Auth.Login).Methods(http.MethodPost)

	// Rotas protegidas: o middleware de autenticação vale para todas as rotas
	// registradas neste subrouter.
	protected := router.NewRoute().Subrouter()
	protected.Use(mux.MiddlewareFunc(middleware.Authenticate(deps.Tokens, deps.Users, deps.Logger)))
	protected.HandleFunc("/me", deps.User.Me).Methods(http.MethodGet)

	return middleware.Chain(router,
		middleware.RequestID(),
		middleware.Logging(deps.Logger),
		middleware.Recovery(deps.Logger),
		middleware.CORS(deps.Config.CORS),
	)
}

// notFound responde rotas inexistentes no padrão da API.
func notFound(w http.ResponseWriter, _ *http.Request) {
	response.Error(w, http.StatusNotFound, "Recurso não encontrado")
}

// methodNotAllowed responde métodos não suportados no padrão da API.
func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	response.Error(w, http.StatusMethodNotAllowed, "Método não permitido para este recurso")
}
