// Package config carrega, valida e disponibiliza as configurações da
// aplicação a partir de variáveis de ambiente (arquivo .env).
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// jwtSecretMinLength é o tamanho mínimo aceito para a chave de assinatura.
const jwtSecretMinLength = 16

// Config é a configuração tipada utilizada por toda a aplicação.
type Config struct {
	App      App
	Database Database
	JWT      JWT
	Log      Log
	CORS     CORS
}

// App reúne as configurações do servidor HTTP.
type App struct {
	Port            string
	Env             string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Addr devolve o endereço de escuta do servidor HTTP.
func (a App) Addr() string {
	return net.JoinHostPort("", a.Port)
}

// IsProduction indica se a aplicação está rodando em ambiente produtivo.
func (a App) IsProduction() bool {
	return strings.EqualFold(a.Env, "production")
}

// Database reúne as configurações de conexão e do pool do PostgreSQL.
type Database struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	AutoMigrate     bool
}

// DSN monta a string de conexão do PostgreSQL.
func (d Database) DSN() string {
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, d.Port),
		Path:     d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}

	return dsn.String()
}

// JWT reúne as configurações de emissão e validação dos tokens.
type JWT struct {
	Secret     string
	Expiration time.Duration
	Issuer     string
}

// Log reúne as configurações do logging estruturado.
type Log struct {
	Level    string
	Dir      string
	ToStdout bool
}

// CORS reúne as configurações do middleware de CORS.
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// Load carrega o arquivo .env (quando existir), valida as variáveis
// obrigatórias e devolve a configuração tipada da aplicação.
//
// Variáveis já definidas no ambiente têm precedência sobre o arquivo .env,
// o que permite sobrescrever valores em contêineres e pipelines de CI/CD.
func Load(files ...string) (*Config, error) {
	if err := loadEnvFiles(files...); err != nil {
		return nil, err
	}

	l := &loader{}

	cfg := &Config{
		App: App{
			Port:            l.port("APP_PORT", "8080"),
			Env:             l.str("APP_ENV", "development"),
			ReadTimeout:     l.duration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    l.duration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     l.duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: l.duration("SHUTDOWN_TIMEOUT", 15*time.Second),
		},
		Database: Database{
			Host:            l.required("DB_HOST"),
			Port:            l.port("DB_PORT", "5432"),
			Name:            l.required("DB_NAME"),
			User:            l.required("DB_USER"),
			Password:        l.required("DB_PASSWORD"),
			SSLMode:         l.sslMode("DB_SSLMODE", "disable"),
			MaxOpenConns:    l.int("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    l.int("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: l.duration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: l.duration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
			AutoMigrate:     l.bool("DB_AUTO_MIGRATE", true),
		},
		JWT: JWT{
			Secret:     l.secret("JWT_SECRET"),
			Expiration: l.duration("JWT_EXPIRATION", 24*time.Hour),
			Issuer:     l.str("JWT_ISSUER", "emeraldfox-api"),
		},
		Log: Log{
			Level:    l.logLevel("LOG_LEVEL", "INFO"),
			Dir:      l.str("LOG_DIR", "logs"),
			ToStdout: l.bool("LOG_TO_STDOUT", true),
		},
		CORS: CORS{
			AllowedOrigins:   l.list("CORS_ALLOWED_ORIGINS", "*"),
			AllowedMethods:   l.list("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS"),
			AllowedHeaders:   l.list("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,X-Request-ID"),
			AllowCredentials: l.bool("CORS_ALLOW_CREDENTIALS", false),
			MaxAge:           l.int("CORS_MAX_AGE", 300),
		},
	}

	if err := l.err(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadEnvFiles carrega os arquivos .env informados. A ausência do arquivo
// padrão não é tratada como erro: a aplicação pode receber a configuração
// diretamente do ambiente.
func loadEnvFiles(files ...string) error {
	if len(files) == 0 {
		files = []string{".env"}
	}

	for _, file := range files {
		if err := godotenv.Load(file); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("carregar arquivo de ambiente %q: %w", file, err)
		}
	}

	return nil
}
