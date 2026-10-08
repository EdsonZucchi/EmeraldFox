package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"emeraldfox/internal/appctx"
	"emeraldfox/pkg/logger"
)

// RequestIDHeader é o cabeçalho usado para propagar o identificador da
// requisição entre serviços.
const RequestIDHeader = "X-Request-ID"

// requestIDMaxLength limita o tamanho aceito para um identificador enviado
// pelo cliente.
const requestIDMaxLength = 64

// RequestID garante que toda requisição tenha um identificador único,
// reaproveitando o enviado pelo cliente quando ele for válido.
//
// Este é o middleware mais externo da aplicação: ele também inicializa o
// contexto de logging, que acumula os atributos (request_id, user_id)
// anexados automaticamente a todos os registros da requisição.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := sanitizeRequestID(r.Header.Get(RequestIDHeader))
			if requestID == "" {
				requestID = newRequestID()
			}

			ctx := appctx.WithRequestID(r.Context(), requestID)
			ctx = logger.NewContext(ctx, slog.String("request_id", requestID))

			w.Header().Set(RequestIDHeader, requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// newRequestID gera um identificador aleatório de 16 caracteres hexadecimais.
func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand não falha na prática; ainda assim a requisição não pode
		// ser interrompida por causa do identificador.
		return "unknown"
	}

	return hex.EncodeToString(buf)
}

// sanitizeRequestID aceita apenas identificadores curtos e com caracteres
// seguros, evitando poluição dos logs e injeção de cabeçalhos na resposta.
func sanitizeRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || len(requestID) > requestIDMaxLength {
		return ""
	}

	for _, r := range requestID {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
		default:
			return ""
		}
	}

	return requestID
}
