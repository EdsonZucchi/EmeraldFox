package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// loader lê variáveis de ambiente acumulando os problemas encontrados, de modo
// que a aplicação reporte todos os erros de configuração de uma só vez.
type loader struct {
	problems []string
}

// err devolve um erro único com todos os problemas encontrados, ou nil.
func (l *loader) err() error {
	if len(l.problems) == 0 {
		return nil
	}

	return fmt.Errorf("configuração inválida:\n  - %s", strings.Join(l.problems, "\n  - "))
}

func (l *loader) fail(format string, args ...any) {
	l.problems = append(l.problems, fmt.Sprintf(format, args...))
}

// lookup devolve o valor da variável já sem espaços nas extremidades.
func lookup(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	return value, true
}

// str devolve o valor da variável ou o padrão informado.
func (l *loader) str(key, fallback string) string {
	if value, ok := lookup(key); ok {
		return value
	}

	return fallback
}

// required devolve o valor da variável e registra um problema quando ausente.
func (l *loader) required(key string) string {
	value, ok := lookup(key)
	if !ok {
		l.fail("%s é obrigatória", key)
	}

	return value
}

// secret valida uma chave secreta obrigatória e o seu tamanho mínimo.
func (l *loader) secret(key string) string {
	value, ok := lookup(key)
	if !ok {
		l.fail("%s é obrigatória", key)
		return ""
	}

	if len(value) < jwtSecretMinLength {
		l.fail("%s deve ter no mínimo %d caracteres", key, jwtSecretMinLength)
	}

	return value
}

// port valida uma porta TCP.
func (l *loader) port(key, fallback string) string {
	value := l.str(key, fallback)

	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > 65535 {
		l.fail("%s deve ser uma porta válida entre 1 e 65535 (recebido %q)", key, value)
	}

	return value
}

// int converte um valor inteiro.
func (l *loader) int(key string, fallback int) int {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		l.fail("%s deve ser um número inteiro (recebido %q)", key, value)
		return fallback
	}

	return number
}

// bool converte um valor booleano (true/false, 1/0, yes/no).
func (l *loader) bool(key string, fallback bool) bool {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}

	switch strings.ToLower(value) {
	case "yes", "y", "on":
		return true
	case "no", "n", "off":
		return false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		l.fail("%s deve ser um booleano (recebido %q)", key, value)
		return fallback
	}

	return parsed
}

// duration converte uma duração no formato aceito por time.ParseDuration
// (ex.: 30s, 15m, 24h).
func (l *loader) duration(key string, fallback time.Duration) time.Duration {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		l.fail("%s deve ser uma duração válida, ex.: 30s, 15m, 24h (recebido %q)", key, value)
		return fallback
	}

	if parsed <= 0 {
		l.fail("%s deve ser maior que zero (recebido %q)", key, value)
		return fallback
	}

	return parsed
}

// list converte uma lista separada por vírgulas.
func (l *loader) list(key, fallback string) []string {
	value := l.str(key, fallback)

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))

	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			items = append(items, part)
		}
	}

	if len(items) == 0 {
		l.fail("%s deve conter ao menos um valor", key)
	}

	return items
}

// sslMode valida o modo SSL aceito pelo driver do PostgreSQL.
func (l *loader) sslMode(key, fallback string) string {
	value := strings.ToLower(l.str(key, fallback))

	switch value {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return value
	default:
		l.fail("%s deve ser disable, allow, prefer, require, verify-ca ou verify-full (recebido %q)", key, value)
		return fallback
	}
}

// logLevel valida o nível mínimo de log.
func (l *loader) logLevel(key, fallback string) string {
	value := strings.ToUpper(l.str(key, fallback))

	switch value {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return value
	default:
		l.fail("%s deve ser DEBUG, INFO, WARN ou ERROR (recebido %q)", key, value)
		return fallback
	}
}
