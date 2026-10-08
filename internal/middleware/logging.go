package middleware

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

// Logging registra uma linha de log estruturado por requisição HTTP.
//
// O nível é escolhido pelo status da resposta: 5xx gera ERROR, 4xx gera WARN e
// os demais casos geram INFO. Os campos request_id e user_id são anexados
// automaticamente pelo contexto de logging.
func Logging(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := newResponseRecorder(w)

			defer func() {
				duration := time.Since(start)
				ctx := r.Context()

				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", recorder.status),
					slog.Float64("duration_ms", millis(duration)),
					slog.String("ip", clientIP(r)),
					slog.String("user_agent", r.UserAgent()),
					slog.Int("bytes", recorder.bytes),
				}

				if query := r.URL.RawQuery; query != "" {
					attrs = append(attrs, slog.String("query", query))
				}

				log.LogAttrs(ctx, levelForStatus(recorder.status), "HTTP Request", attrs...)
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}

// levelForStatus mapeia o status HTTP para o nível de log correspondente.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// millis converte a duração em milissegundos com duas casas decimais.
func millis(d time.Duration) float64 {
	return math.Round(float64(d.Microseconds())/10) / 100
}

// clientIP identifica o IP de origem da requisição.
//
// Quando a API roda atrás de um proxy reverso confiável, o IP real chega em
// X-Forwarded-For ou X-Real-IP; caso contrário, usa-se o endereço da conexão.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first, _, found := strings.Cut(forwarded, ","); found {
			forwarded = first
		}
		if ip := strings.TrimSpace(forwarded); ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
