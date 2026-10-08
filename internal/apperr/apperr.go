// Package apperr define o erro de aplicação usado para transportar, de forma
// consistente, o status HTTP e a mensagem segura que deve chegar ao cliente,
// preservando o erro original e o stack trace apenas para os logs.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// internalMessage é a mensagem devolvida ao cliente em falhas inesperadas.
// Detalhes internos nunca são expostos: eles ficam apenas nos logs.
const internalMessage = "Erro interno do servidor"

// Error é o erro de aplicação. Message é sempre segura para exposição pública;
// Err e Stack existem exclusivamente para observabilidade.
type Error struct {
	Status  int
	Message string
	Err     error
	Stack   string
}

// Error implementa a interface error.
func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}

	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// Unwrap permite o uso de errors.Is e errors.As sobre o erro original.
func (e *Error) Unwrap() error {
	return e.Err
}

// IsInternal indica se o erro representa uma falha do servidor (5xx).
func (e *Error) IsInternal() bool {
	return e.Status >= http.StatusInternalServerError
}

// New cria um erro de aplicação com status e mensagem públicos.
func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

// Wrap cria um erro de aplicação preservando a causa original.
func Wrap(err error, status int, message string) *Error {
	appErr := &Error{Status: status, Message: message, Err: err}
	if appErr.IsInternal() {
		appErr.Stack = captureStack(2)
	}

	return appErr
}

// BadRequest indica requisição malformada.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message)
}

// Validation indica dados de entrada inválidos.
func Validation(message string) *Error {
	return New(http.StatusUnprocessableEntity, message)
}

// Unauthorized indica ausência ou invalidez de credenciais.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, message)
}

// Forbidden indica credencial válida sem permissão para o recurso.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, message)
}

// NotFound indica recurso inexistente.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, message)
}

// Conflict indica conflito com o estado atual do recurso.
func Conflict(message string) *Error {
	return New(http.StatusConflict, message)
}

// Internal cria um erro interno com stack trace, expondo ao cliente apenas
// uma mensagem genérica.
func Internal(err error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Message: internalMessage,
		Err:     err,
		Stack:   captureStack(2),
	}
}

// Internalf cria um erro interno anotando a operação de origem.
func Internalf(err error, format string, args ...any) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Message: internalMessage,
		Err:     fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err),
		Stack:   captureStack(2),
	}
}

// From converte qualquer erro em *Error. Erros desconhecidos são tratados como
// falhas internas para que nenhum detalhe vaze na resposta HTTP.
func From(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	return &Error{
		Status:  http.StatusInternalServerError,
		Message: internalMessage,
		Err:     err,
		Stack:   captureStack(2),
	}
}

// captureStack devolve o stack trace formatado a partir do chamador indicado.
func captureStack(skip int) string {
	const depth = 32

	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip+1, pcs)
	if n == 0 {
		return ""
	}

	var (
		builder strings.Builder
		frames  = runtime.CallersFrames(pcs[:n])
	)

	for {
		frame, more := frames.Next()
		fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}

	return builder.String()
}
