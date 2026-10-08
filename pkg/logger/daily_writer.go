package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// dateLayout é o formato dos nomes dos arquivos de log (YYYY-MM-DD.log).
const dateLayout = "2006-01-02"

// DailyWriter grava em um arquivo por dia dentro de um diretório, criando um
// novo arquivo automaticamente na virada do dia (rotação diária).
//
// A implementação é segura para uso concorrente.
type DailyWriter struct {
	dir string

	mu   sync.Mutex
	file *os.File
	day  string
}

// NewDailyWriter cria o writer e garante a existência do diretório informado.
func NewDailyWriter(dir string) (*DailyWriter, error) {
	if dir == "" {
		return nil, fmt.Errorf("diretório de logs não informado")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("criar diretório de logs %q: %w", dir, err)
	}

	w := &DailyWriter{dir: dir}
	if err := w.rotate(currentDay()); err != nil {
		return nil, err
	}

	return w, nil
}

// Write grava os bytes no arquivo do dia corrente, rotacionando se necessário.
func (w *DailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if day := currentDay(); day != w.day {
		if err := w.rotate(day); err != nil {
			return 0, err
		}
	}

	return w.file.Write(p)
}

// Close fecha o arquivo aberto.
func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	err := w.file.Close()
	w.file = nil

	return err
}

// Path devolve o caminho do arquivo de log em uso.
func (w *DailyWriter) Path() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return filepath.Join(w.dir, w.day+".log")
}

// rotate fecha o arquivo atual e abre o arquivo correspondente ao dia informado.
// Deve ser chamado com o mutex já adquirido (ou antes da publicação do writer).
func (w *DailyWriter) rotate(day string) error {
	path := filepath.Join(w.dir, day+".log")

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("abrir arquivo de log %q: %w", path, err)
	}

	if w.file != nil {
		_ = w.file.Close()
	}

	w.file = file
	w.day = day

	return nil
}

// currentDay devolve o dia corrente em UTC, mesma referência usada nos
// timestamps dos registros.
func currentDay() string {
	return time.Now().UTC().Format(dateLayout)
}
