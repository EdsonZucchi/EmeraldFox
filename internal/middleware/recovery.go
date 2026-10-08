package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"emeraldfox/internal/response"
)

// Recovery converte panics em respostas 500 sem derrubar o servidor.
//
// O stack trace completo vai para o log; o cliente recebe apenas a mensagem
// genérica de erro interno.
func Recovery(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				// Sinaliza ao servidor HTTP o abandono deliberado da conexão.
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}

				log.ErrorContext(r.Context(), "Panic recuperado ao processar requisição",
					slog.String("error", fmt.Sprintf("%v", recovered)),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)

				response.Error(w, http.StatusInternalServerError, "Erro interno do servidor")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
