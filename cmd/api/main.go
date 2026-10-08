// Command api é o ponto de entrada da REST API do EmeraldFox.
//
// Responsabilidades: carregar a configuração, inicializar as dependências
// (logger, banco de dados, repositórios, serviços e handlers), servir o HTTP e
// encerrar tudo de forma controlada.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"emeraldfox/internal/auth"
	"emeraldfox/internal/config"
	"emeraldfox/internal/database"
	"emeraldfox/internal/handlers"
	"emeraldfox/internal/repository"
	"emeraldfox/internal/routes"
	"emeraldfox/internal/service"
	"emeraldfox/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		// A aplicação pode falhar antes do logger existir, então o erro fatal
		// também é reportado no stderr.
		fmt.Fprintf(os.Stderr, "falha ao iniciar a aplicação: %v\n", err)
		os.Exit(1)
	}
}

// run concentra a inicialização para que todos os defers rodem antes da saída.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	appLogger, err := logger.New(logger.Config{
		Level:    cfg.Log.Level,
		Dir:      cfg.Log.Dir,
		ToStdout: cfg.Log.ToStdout,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = appLogger.Close()
	}()

	log := appLogger.Logger

	// Disponibiliza o logger estruturado também para o slog padrão, evitando
	// que bibliotecas de terceiros escrevam em formato divergente.
	slog.SetDefault(log)

	logDir, err := filepath.Abs(cfg.Log.Dir)
	if err != nil {
		logDir = cfg.Log.Dir
	}

	// Encerramento controlado ao receber SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.InfoContext(ctx, "Iniciando aplicação",
		slog.String("env", cfg.App.Env),
		slog.String("port", cfg.App.Port),
		slog.String("log_level", cfg.Log.Level),
		slog.String("log_dir", logDir),
	)

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.ErrorContext(ctx, "Não foi possível conectar ao banco de dados", slog.String("error", err.Error()))
		return err
	}
	defer closeDatabase(ctx, db, log)

	log.InfoContext(ctx, "Conectado ao PostgreSQL",
		slog.String("host", cfg.Database.Host),
		slog.String("database", cfg.Database.Name),
		slog.Int("max_open_conns", cfg.Database.MaxOpenConns),
	)

	if cfg.Database.AutoMigrate {
		if err := database.Migrate(ctx, db, log); err != nil {
			log.ErrorContext(ctx, "Não foi possível aplicar as migrations", slog.String("error", err.Error()))
			return err
		}
	}

	// Injeção de dependências: repositório -> serviço -> handlers -> rotas.
	userRepository := repository.NewUserRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.Expiration, cfg.JWT.Issuer)
	userService := service.NewUserService(userRepository, tokenManager, log)

	router := routes.New(routes.Dependencies{
		Config: cfg,
		Logger: log,
		Tokens: tokenManager,
		Users:  userService,
		Auth:   handlers.NewAuthHandler(userService, log),
		User:   handlers.NewUserHandler(log),
		Health: handlers.NewHealthHandler(db, log),
	})

	server := &http.Server{
		Addr:         cfg.App.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	serverErr := make(chan error, 1)

	go func() {
		log.InfoContext(ctx, "Servidor HTTP disponível", slog.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("servidor HTTP: %w", err)
		}
	}()

	select {
	case err := <-serverErr:
		log.ErrorContext(ctx, "Servidor HTTP encerrado com erro", slog.String("error", err.Error()))
		return err
	case <-ctx.Done():
		stop()
		log.InfoContext(context.Background(), "Sinal de encerramento recebido, finalizando a aplicação")
	}

	return shutdown(server, cfg.App.ShutdownTimeout, log)
}

// shutdown encerra o servidor aguardando a conclusão das requisições em curso.
func shutdown(server *http.Server, timeout time.Duration, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.ErrorContext(ctx, "Encerramento do servidor excedeu o tempo limite", slog.String("error", err.Error()))
		_ = server.Close()

		return fmt.Errorf("encerrar servidor HTTP: %w", err)
	}

	log.InfoContext(ctx, "Aplicação encerrada com sucesso")

	return nil
}

// closeDatabase fecha o pool de conexões registrando eventuais falhas.
func closeDatabase(ctx context.Context, db *sql.DB, log *slog.Logger) {
	if err := db.Close(); err != nil {
		log.ErrorContext(ctx, "Falha ao fechar a conexão com o banco de dados", slog.String("error", err.Error()))
		return
	}

	log.InfoContext(context.Background(), "Conexão com o banco de dados encerrada")
}
