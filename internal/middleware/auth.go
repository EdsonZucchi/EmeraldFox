package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"emeraldfox/internal/appctx"
	"emeraldfox/internal/auth"
	"emeraldfox/internal/models"
	"emeraldfox/internal/response"
	"emeraldfox/pkg/logger"

	"github.com/google/uuid"
)

// bearerPrefix é o esquema de autorização aceito.
const bearerPrefix = "Bearer "

// UserLoader carrega o usuário autenticado a partir do identificador contido
// no token.
type UserLoader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// Authenticate valida o JWT, carrega o usuário no banco e o disponibiliza no
// contexto da requisição.
//
// Nenhum handler deve validar tokens manualmente: o usuário autenticado é
// obtido exclusivamente por appctx.User.
func Authenticate(tokens *auth.TokenManager, users UserLoader, log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			token, err := bearerToken(r)
			if err != nil {
				unauthorized(ctx, w, log, "Token de autenticação não informado", err)
				return
			}

			claims, err := tokens.ValidateToken(token)
			if err != nil {
				if errors.Is(err, auth.ErrExpiredToken) {
					unauthorized(ctx, w, log, "Token expirado", err)
					return
				}
				unauthorized(ctx, w, log, "Token inválido", err)
				return
			}

			userID, err := claims.UserID()
			if err != nil {
				unauthorized(ctx, w, log, "Token inválido", err)
				return
			}

			user, err := users.GetByID(ctx, userID)
			if err != nil {
				// O token é válido, mas o usuário não existe mais: tratado como
				// credencial inválida para não revelar o estado do cadastro.
				log.WarnContext(ctx, "Usuário do token não pôde ser carregado",
					slog.String("user_id", userID.String()),
					slog.String("error", err.Error()),
				)
				response.Error(w, http.StatusUnauthorized, "Token inválido")

				return
			}

			ctx = appctx.WithUser(ctx, user)
			logger.AddAttrs(ctx, slog.String("user_id", user.ID.String()))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extrai o token do cabeçalho Authorization.
func bearerToken(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", errors.New("cabeçalho Authorization ausente")
	}

	if len(header) <= len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", errors.New("cabeçalho Authorization fora do formato \"Bearer <token>\"")
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", errors.New("token vazio")
	}

	return token, nil
}

// unauthorized registra a causa real da recusa e devolve ao cliente apenas a
// mensagem pública correspondente.
func unauthorized(ctx context.Context, w http.ResponseWriter, log *slog.Logger, message string, err error) {
	log.WarnContext(ctx, "Requisição não autenticada",
		slog.String("reason", message),
		slog.String("error", err.Error()),
	)

	w.Header().Set("WWW-Authenticate", `Bearer realm="emeraldfox"`)
	response.Error(w, http.StatusUnauthorized, message)
}
