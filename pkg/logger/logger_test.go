package logger_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"emeraldfox/pkg/logger"
)

func TestLogsSaoGravadosEmJSONLinesNoArquivoDoDia(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")

	appLogger, err := logger.New(logger.Config{Level: "INFO", Dir: dir})
	if err != nil {
		t.Fatalf("New devolveu erro: %v", err)
	}
	defer func() {
		_ = appLogger.Close()
	}()

	ctx := logger.NewContext(context.Background(), slog.String("request_id", "4c7c4e96d4f14cb0"))
	logger.AddAttrs(ctx, slog.String("user_id", "5c8df77e-56dc-4d44-a090-53dcb7f5f38e"))

	appLogger.InfoContext(ctx, "HTTP Request", slog.String("method", "GET"), slog.Int("status", 200))
	appLogger.DebugContext(ctx, "Não deve aparecer: abaixo do nível mínimo")

	if err := appLogger.Close(); err != nil {
		t.Fatalf("Close devolveu erro: %v", err)
	}

	path := filepath.Join(dir, time.Now().UTC().Format("2006-01-02")+".log")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("arquivo de log %q não foi criado: %v", path, err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Fatalf("esperado 1 linha de log, recebido %d: %q", len(lines), content)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("linha de log não é um JSON válido: %v", err)
	}

	expected := map[string]any{
		"level":      "INFO",
		"message":    "HTTP Request",
		"method":     "GET",
		"request_id": "4c7c4e96d4f14cb0",
		"user_id":    "5c8df77e-56dc-4d44-a090-53dcb7f5f38e",
	}

	for key, want := range expected {
		if got := record[key]; got != want {
			t.Errorf("campo %q = %v, esperado %v", key, got, want)
		}
	}

	timestamp, ok := record["timestamp"].(string)
	if !ok {
		t.Fatalf("campo timestamp ausente no registro: %v", record)
	}

	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Errorf("timestamp %q não está em RFC3339: %v", timestamp, err)
	}
}

func TestNewCriaDiretorioInexistente(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nivel-1", "nivel-2", "logs")

	appLogger, err := logger.New(logger.Config{Level: "DEBUG", Dir: dir})
	if err != nil {
		t.Fatalf("New devolveu erro: %v", err)
	}
	defer func() {
		_ = appLogger.Close()
	}()

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("diretório de logs não foi criado: %v", err)
	}
}

func TestNewRejeitaNivelInvalido(t *testing.T) {
	if _, err := logger.New(logger.Config{Level: "TRACE", Dir: t.TempDir()}); err == nil {
		t.Fatal("esperado erro para nível de log inválido")
	}
}

func TestDailyWriterAbreArquivoDoDiaEReaproveitaExistente(t *testing.T) {
	dir := t.TempDir()

	writer, err := logger.NewDailyWriter(dir)
	if err != nil {
		t.Fatalf("NewDailyWriter devolveu erro: %v", err)
	}

	if _, err := writer.Write([]byte("primeira linha\n")); err != nil {
		t.Fatalf("Write devolveu erro: %v", err)
	}

	path := writer.Path()

	if err := writer.Close(); err != nil {
		t.Fatalf("Close devolveu erro: %v", err)
	}

	// Reabrir o writer deve preservar o conteúdo já gravado (modo append).
	reopened, err := logger.NewDailyWriter(dir)
	if err != nil {
		t.Fatalf("NewDailyWriter devolveu erro na reabertura: %v", err)
	}
	defer func() {
		_ = reopened.Close()
	}()

	if _, err := reopened.Write([]byte("segunda linha\n")); err != nil {
		t.Fatalf("Write devolveu erro: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler arquivo de log: %v", err)
	}

	if got := strings.Count(string(content), "\n"); got != 2 {
		t.Errorf("linhas gravadas = %d, esperado 2 (conteúdo: %q)", got, content)
	}

	if filepath.Base(path) != time.Now().UTC().Format("2006-01-02")+".log" {
		t.Errorf("nome do arquivo = %q, esperado YYYY-MM-DD.log do dia corrente", filepath.Base(path))
	}
}
