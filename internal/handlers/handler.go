// Package handlers implementa a camada HTTP: leitura da requisição, chamada
// do serviço correspondente e escrita da resposta padronizada.
//
// Handlers não contêm regra de negócio, não acessam o banco diretamente e
// nunca validam tokens: o usuário autenticado vem sempre do contexto.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"emeraldfox/internal/apperr"
	"emeraldfox/internal/response"
)

// maxRequestBody limita o tamanho do corpo aceito nas requisições JSON.
const maxRequestBody = 1 << 20 // 1 MiB

// base concentra o comportamento comum a todos os handlers.
type base struct {
	log *slog.Logger
}

// fail registra o erro e devolve a resposta padronizada correspondente.
//
// Erros internos (5xx) são logados com a causa original e o stack trace, mas o
// cliente recebe apenas a mensagem genérica definida em apperr.
func (b base) fail(w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperr.From(err)
	ctx := r.Context()

	if appErr.IsInternal() {
		attrs := []slog.Attr{
			slog.String("error", appErr.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		}
		if appErr.Stack != "" {
			attrs = append(attrs, slog.String("stack", appErr.Stack))
		}

		b.log.LogAttrs(ctx, slog.LevelError, "Falha ao processar requisição", attrs...)
	} else {
		b.log.LogAttrs(ctx, slog.LevelDebug, "Requisição rejeitada",
			slog.Int("status", appErr.Status),
			slog.String("error", appErr.Error()),
		)
	}

	response.Error(w, appErr.Status, appErr.Message)
}

// decode lê e valida o corpo JSON da requisição.
func (b base) decode(w http.ResponseWriter, r *http.Request, dst any) error {
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		if mediaType, _, _ := strings.Cut(contentType, ";"); !strings.EqualFold(strings.TrimSpace(mediaType), "application/json") {
			return apperr.New(http.StatusUnsupportedMediaType, "O corpo da requisição deve ser application/json")
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return decodeError(err)
	}

	// Garante que o corpo contenha um único documento JSON.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperr.BadRequest("O corpo da requisição deve conter um único objeto JSON")
	}

	return nil
}

// decodeError traduz as falhas de desserialização em mensagens públicas.
func decodeError(err error) error {
	var (
		syntaxErr       *json.SyntaxError
		unmarshalErr    *json.UnmarshalTypeError
		maxBytesErr     *http.MaxBytesError
		unknownFieldMsg = "json: unknown field "
	)

	switch {
	case errors.Is(err, io.EOF):
		return apperr.BadRequest("O corpo da requisição é obrigatório")
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return apperr.BadRequest("O corpo da requisição não é um JSON válido")
	case errors.As(err, &unmarshalErr):
		return apperr.BadRequest("O campo \"" + unmarshalErr.Field + "\" possui um tipo inválido")
	case errors.As(err, &maxBytesErr):
		return apperr.New(http.StatusRequestEntityTooLarge, "O corpo da requisição excede o tamanho máximo permitido")
	case strings.HasPrefix(err.Error(), unknownFieldMsg):
		field := strings.Trim(strings.TrimPrefix(err.Error(), unknownFieldMsg), `"`)
		return apperr.BadRequest("O campo \"" + field + "\" não é aceito nesta requisição")
	default:
		return apperr.Wrap(err, http.StatusBadRequest, "Não foi possível ler o corpo da requisição")
	}
}
