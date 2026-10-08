// Package repository isola o acesso aos dados. As camadas superiores conhecem
// apenas os erros sentinela declarados aqui, nunca detalhes do driver SQL.
package repository

import "errors"

var (
	// ErrNotFound indica que o registro consultado não existe.
	ErrNotFound = errors.New("registro não encontrado")
	// ErrDuplicate indica violação de uma restrição de unicidade.
	ErrDuplicate = errors.New("registro duplicado")
)
