package logger

import (
	"context"
	"log/slog"
	"sync"
)

// attrsKey é a chave (não exportada) usada para guardar os atributos de log
// no context.Context.
type attrsKey struct{}

// attrStore acumula atributos durante o ciclo de vida de uma requisição.
//
// O ponteiro é armazenado no contexto uma única vez (pelo middleware mais
// externo); middlewares e handlers subsequentes apenas adicionam atributos,
// o que permite que camadas externas — como o middleware de logging — enxerguem
// dados descobertos por camadas internas, como o user_id resolvido pela
// autenticação.
type attrStore struct {
	mu    sync.RWMutex
	attrs []slog.Attr
}

func (s *attrStore) add(attrs ...slog.Attr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attrs = append(s.attrs, attrs...)
}

func (s *attrStore) snapshot() []slog.Attr {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.attrs) == 0 {
		return nil
	}

	out := make([]slog.Attr, len(s.attrs))
	copy(out, s.attrs)

	return out
}

// NewContext devolve um contexto capaz de acumular atributos de log. Todos os
// registros emitidos com esse contexto recebem automaticamente os atributos
// acumulados (request_id, user_id, ...).
func NewContext(ctx context.Context, attrs ...slog.Attr) context.Context {
	store := &attrStore{}
	store.add(attrs...)

	return context.WithValue(ctx, attrsKey{}, store)
}

// AddAttrs adiciona atributos ao contexto criado por NewContext. É seguro
// chamar em contextos sem store: nesse caso a chamada é ignorada.
func AddAttrs(ctx context.Context, attrs ...slog.Attr) {
	if store, ok := ctx.Value(attrsKey{}).(*attrStore); ok {
		store.add(attrs...)
	}
}

// Attrs devolve uma cópia dos atributos acumulados no contexto.
func Attrs(ctx context.Context) []slog.Attr {
	if store, ok := ctx.Value(attrsKey{}).(*attrStore); ok {
		return store.snapshot()
	}

	return nil
}

// contextHandler enriquece cada registro com os atributos presentes no
// contexto, dispensando a repetição manual de request_id/user_id em cada log.
type contextHandler struct {
	slog.Handler
}

func newContextHandler(h slog.Handler) slog.Handler {
	return contextHandler{Handler: h}
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if attrs := Attrs(ctx); len(attrs) > 0 {
		record = record.Clone()
		record.AddAttrs(attrs...)
	}

	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}
