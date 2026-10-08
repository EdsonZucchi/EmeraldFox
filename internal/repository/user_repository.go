package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"emeraldfox/internal/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// uniqueViolationCode é o SQLSTATE do PostgreSQL para violação de unicidade.
const uniqueViolationCode = "23505"

// UserRepository implementa o acesso aos dados de usuários.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository cria o repositório de usuários.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create persiste um novo usuário.
//
// Devolve ErrDuplicate quando o e-mail já estiver cadastrado.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	const query = `
		INSERT INTO users (id, name, email, created_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Email, user.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("criar usuário: %w", ErrDuplicate)
		}
		return fmt.Errorf("criar usuário: %w", err)
	}

	return nil
}

// GetByID busca um usuário pelo identificador.
//
// Devolve ErrNotFound quando o usuário não existir.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	const query = `
		SELECT id, name, email, created_at
		FROM users
		WHERE id = $1`

	return r.queryUser(ctx, "buscar usuário por id", query, id)
}

// GetByEmail busca um usuário pelo e-mail.
//
// Devolve ErrNotFound quando o usuário não existir.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const query = `
		SELECT id, name, email, created_at
		FROM users
		WHERE email = $1`

	return r.queryUser(ctx, "buscar usuário por e-mail", query, email)
}

// ExistsByEmail informa se já existe um usuário com o e-mail informado.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar e-mail existente: %w", err)
	}

	return exists, nil
}

// queryUser executa uma consulta que devolve um único usuário.
func (r *UserRepository) queryUser(ctx context.Context, operation, query string, args ...any) (*models.User, error) {
	var user models.User

	err := r.db.QueryRowContext(ctx, query, args...).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", operation, ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	return &user, nil
}

// isUniqueViolation identifica violações de unicidade reportadas pelo driver.
func isUniqueViolation(err error) bool {
	var pgErr *pq.Error

	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
