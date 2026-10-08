// Package logger concentra a configuração do logging estruturado da aplicação.
//
// Os logs são emitidos em JSON Lines (um objeto JSON por linha) através do
// pacote padrão log/slog, gravados em arquivos com rotação diária e,
// opcionalmente, espelhados no stdout.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Chaves padronizadas para os campos obrigatórios de todo log.
const (
	keyTimestamp = "timestamp"
	keyMessage   = "message"
)

// Config reúne as opções de inicialização do logger.
type Config struct {
	// Level é o nível mínimo registrado (DEBUG, INFO, WARN ou ERROR).
	Level string
	// Dir é o diretório onde os arquivos diários são gravados.
	Dir string
	// ToStdout espelha os logs no console além de gravá-los em arquivo.
	ToStdout bool
	// AddSource inclui arquivo/linha de origem em cada registro.
	AddSource bool
}

// Logger é o logger da aplicação junto do recurso de arquivo que o alimenta.
type Logger struct {
	*slog.Logger

	writer *DailyWriter
}

// New cria o logger da aplicação.
//
// O diretório de logs é criado automaticamente caso não exista. O chamador é
// responsável por invocar Close durante o encerramento da aplicação.
func New(cfg Config) (*Logger, error) {
	level, err := ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	writer, err := NewDailyWriter(cfg.Dir)
	if err != nil {
		return nil, err
	}

	var out io.Writer = writer
	if cfg.ToStdout {
		out = io.MultiWriter(writer, os.Stdout)
	}

	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level:       level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: replaceAttr,
	})

	return &Logger{
		Logger: slog.New(newContextHandler(handler)),
		writer: writer,
	}, nil
}

// Close libera o arquivo de log atualmente aberto.
func (l *Logger) Close() error {
	if l == nil || l.writer == nil {
		return nil
	}
	return l.writer.Close()
}

// ParseLevel converte a representação textual de um nível de log.
func ParseLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "", "INFO":
		return slog.LevelInfo, nil
	case "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("nível de log inválido: %q (use DEBUG, INFO, WARN ou ERROR)", level)
	}
}

// replaceAttr normaliza os campos obrigatórios de todo registro: "timestamp"
// em UTC/RFC3339 e "message" como chave da mensagem.
func replaceAttr(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return attr
	}

	switch attr.Key {
	case slog.TimeKey:
		return slog.String(keyTimestamp, attr.Value.Time().UTC().Format(time.RFC3339))
	case slog.MessageKey:
		return slog.String(keyMessage, attr.Value.String())
	default:
		return attr
	}
}
