// Package database cuida da conexão com o PostgreSQL e da aplicação das
// migrations do schema.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"emeraldfox/internal/config"

	// Driver do PostgreSQL utilizado pelo database/sql.
	_ "github.com/lib/pq"
)

// pingTimeout limita a verificação de disponibilidade feita na inicialização.
const pingTimeout = 5 * time.Second

// Connect abre o pool de conexões com o PostgreSQL e valida a conectividade
// com um ping. O chamador é responsável por fechar o *sql.DB devolvido.
func Connect(ctx context.Context, cfg config.Database) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("abrir conexão com o banco de dados: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("conectar em %s:%s/%s: %w", cfg.Host, cfg.Port, cfg.Name, err)
	}

	return db, nil
}
