package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// responseRecorder envolve o http.ResponseWriter para capturar o status e o
// tamanho da resposta, informações necessárias ao log de cada requisição.
type responseRecorder struct {
	http.ResponseWriter

	status      int
	bytes       int
	wroteHeader bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}

	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	n, err := r.ResponseWriter.Write(b)
	r.bytes += n

	return n, err
}

// Flush mantém o suporte a respostas em streaming.
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack mantém o suporte a protocolos que assumem a conexão (ex.: WebSocket).
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.Hijacker não suportado pelo ResponseWriter")
	}

	return hijacker.Hijack()
}
