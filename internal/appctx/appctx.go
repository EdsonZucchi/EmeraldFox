// Package appctx padroniza a leitura e a escrita dos valores por requisição
// no context.Context (usuário autenticado e request_id).
//
// Ele evita que middlewares e handlers dependam uns dos outros apenas para
// compartilhar esses valores.
package appctx

import (
	"context"

	"emeraldfox/internal/models"
)

type (
	userKey      struct{}
	requestIDKey struct{}
)

// WithUser devolve um contexto contendo o usuário autenticado.
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

// User devolve o usuário autenticado presente no contexto.
func User(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey{}).(*models.User)
	if !ok || user == nil {
		return nil, false
	}

	return user, true
}

// WithRequestID devolve um contexto contendo o identificador da requisição.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID devolve o identificador da requisição presente no contexto.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)

	return requestID
}
