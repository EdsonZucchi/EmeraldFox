// Package response padroniza o corpo de todas as respostas da API.
//
// Sucesso: {"success": true, "data": {...}}
// Erro:    {"success": false, "message": "..."}
package response

import (
	"encoding/json"
	"net/http"
)

// successBody é o envelope das respostas bem-sucedidas.
type successBody struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// errorBody é o envelope das respostas de erro.
type errorBody struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Success escreve uma resposta de sucesso com o status informado.
func Success(w http.ResponseWriter, status int, data any) {
	if data == nil {
		data = struct{}{}
	}

	write(w, status, successBody{Success: true, Data: data})
}

// OK escreve uma resposta de sucesso com status 200.
func OK(w http.ResponseWriter, data any) {
	Success(w, http.StatusOK, data)
}

// Created escreve uma resposta de sucesso com status 201.
func Created(w http.ResponseWriter, data any) {
	Success(w, http.StatusCreated, data)
}

// Error escreve uma resposta de erro com o status e a mensagem informados.
//
// A mensagem deve ser sempre segura para exposição pública: detalhes internos
// pertencem exclusivamente aos logs.
func Error(w http.ResponseWriter, status int, message string) {
	write(w, status, errorBody{Success: false, Message: message})
}

// write serializa o corpo antes de enviar o cabeçalho, de modo que uma falha
// de serialização ainda permita responder com um status de erro coerente.
func write(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		payload = []byte(`{"success":false,"message":"Erro interno do servidor"}`)
		status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	_, _ = w.Write(append(payload, '\n'))
}
