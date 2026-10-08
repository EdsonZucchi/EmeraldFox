// Package middleware reúne os middlewares HTTP da aplicação. Todos são
// independentes entre si e podem ser combinados em qualquer ordem através de
// Chain ou do Use do gorilla/mux.
package middleware

import "net/http"

// Middleware é a assinatura padrão dos middlewares da aplicação.
type Middleware func(http.Handler) http.Handler

// Chain compõe os middlewares em torno do handler informado. O primeiro
// middleware da lista é o mais externo, isto é, o primeiro a receber a
// requisição e o último a ver a resposta.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
