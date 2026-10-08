package service

import (
	"net/mail"
	"strings"
	"unicode/utf8"

	"emeraldfox/internal/apperr"
)

// Limites alinhados às colunas da tabela users.
const (
	nameMinLength  = 2
	nameMaxLength  = 120
	emailMaxLength = 255
)

// validateName normaliza e valida o nome informado.
func validateName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")

	switch {
	case name == "":
		return "", apperr.Validation("O nome é obrigatório")
	case utf8.RuneCountInString(name) < nameMinLength:
		return "", apperr.Validation("O nome deve ter no mínimo 2 caracteres")
	case utf8.RuneCountInString(name) > nameMaxLength:
		return "", apperr.Validation("O nome deve ter no máximo 120 caracteres")
	}

	return name, nil
}

// validateEmail normaliza e valida o e-mail informado.
//
// A normalização em minúsculas garante que o e-mail seja tratado de forma
// consistente no cadastro, no login e no índice único do banco.
func validateEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	switch {
	case email == "":
		return "", apperr.Validation("O e-mail é obrigatório")
	case len(email) > emailMaxLength:
		return "", apperr.Validation("O e-mail deve ter no máximo 255 caracteres")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", apperr.Validation("O e-mail informado é inválido")
	}

	return email, nil
}
