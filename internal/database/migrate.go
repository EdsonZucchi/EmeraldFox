package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

// migrationsFS embute os arquivos .sql no binário, evitando dependência de
// arquivos externos no ambiente de execução.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// createMigrationsTable registra quais versões já foram aplicadas.
const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT        PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// Migrate aplica, em ordem, as migrations ainda não executadas.
//
// Cada migration roda dentro de uma transação: em caso de falha nada é
// persistido e a aplicação não sobe com o schema pela metade.
func Migrate(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	if _, err := db.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("criar tabela de controle de migrations: %w", err)
	}

	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	files, err := migrationFiles()
	if err != nil {
		return err
	}

	pending := 0

	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		if _, ok := applied[version]; ok {
			continue
		}

		statements, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("ler migration %s: %w", file, err)
		}

		if err := applyMigration(ctx, db, version, string(statements)); err != nil {
			return err
		}

		pending++
		log.InfoContext(ctx, "Migration aplicada", slog.String("version", version))
	}

	if pending == 0 {
		log.InfoContext(ctx, "Schema do banco de dados já está atualizado")
	}

	return nil
}

// applyMigration executa uma migration e registra a sua versão na mesma
// transação.
func applyMigration(ctx context.Context, db *sql.DB, version, statements string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação da migration %s: %w", version, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, statements); err != nil {
		return fmt.Errorf("executar migration %s: %w", version, err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return fmt.Errorf("registrar migration %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar migration %s: %w", version, err)
	}

	return nil
}

// appliedMigrations devolve o conjunto de versões já aplicadas.
func appliedMigrations(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("consultar migrations aplicadas: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	applied := make(map[string]struct{})

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("ler migrations aplicadas: %w", err)
		}
		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ler migrations aplicadas: %w", err)
	}

	return applied, nil
}

// migrationFiles devolve os nomes dos arquivos de migration em ordem
// lexicográfica, que corresponde à ordem de execução.
func migrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("listar migrations: %w", err)
	}

	files := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)

	return files, nil
}
