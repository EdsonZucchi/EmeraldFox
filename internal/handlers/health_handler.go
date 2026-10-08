package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"emeraldfox/internal/response"
)

// healthTimeout limita a verificação de disponibilidade do banco.
const healthTimeout = 2 * time.Second

// Pinger verifica a disponibilidade de uma dependência externa.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// HealthHandler expõe o endpoint de verificação de saúde da API.
type HealthHandler struct {
	base

	db Pinger
}

// NewHealthHandler cria o handler de health check.
func NewHealthHandler(db Pinger, log *slog.Logger) *HealthHandler {
	return &HealthHandler{base: base{log: log}, db: db}
}

// healthResponse descreve o estado da API e de suas dependências.
type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// Health trata GET /health.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		h.log.ErrorContext(ctx, "Health check falhou", slog.String("error", err.Error()))
		response.Error(w, http.StatusServiceUnavailable, "Serviço temporariamente indisponível")

		return
	}

	response.OK(w, healthResponse{Status: "ok", Database: "up"})
}
