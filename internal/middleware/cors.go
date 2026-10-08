package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"emeraldfox/internal/config"
)

// CORS aplica a política de compartilhamento entre origens e responde às
// requisições de verificação prévia (preflight).
func CORS(cfg config.CORS) Middleware {
	var (
		allowedMethods = strings.Join(cfg.AllowedMethods, ", ")
		allowedHeaders = strings.Join(cfg.AllowedHeaders, ", ")
		maxAge         = strconv.Itoa(cfg.MaxAge)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Respostas variam conforme a origem: o cabeçalho evita que caches
			// compartilhem a resposta de uma origem com outra.
			w.Header().Add("Vary", "Origin")

			if allowed, ok := allowOrigin(cfg, origin); ok {
				w.Header().Set("Access-Control-Allow-Origin", allowed)

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if isPreflight(r) {
					w.Header().Add("Vary", "Access-Control-Request-Method")
					w.Header().Add("Vary", "Access-Control-Request-Headers")
					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
					w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
					w.Header().Set("Access-Control-Max-Age", maxAge)
					w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)

					w.WriteHeader(http.StatusNoContent)

					return
				}

				w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allowOrigin decide se a origem é permitida e qual valor deve ser devolvido
// no cabeçalho Access-Control-Allow-Origin.
func allowOrigin(cfg config.CORS, origin string) (string, bool) {
	if origin == "" {
		return "", false
	}

	for _, allowed := range cfg.AllowedOrigins {
		if allowed == "*" {
			// Com credenciais o curinga não é aceito pelos navegadores: nesse
			// caso a própria origem da requisição é ecoada.
			if cfg.AllowCredentials {
				return origin, true
			}
			return "*", true
		}

		if strings.EqualFold(allowed, origin) {
			return origin, true
		}
	}

	return "", false
}

// isPreflight identifica a requisição de verificação prévia do CORS.
func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}
