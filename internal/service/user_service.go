// Package service concentra as regras de negócio da aplicação. É a única
// camada que orquestra repositórios, autenticação e validações; handlers
// cuidam apenas do protocolo HTTP.
package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"emeraldfox/internal/apperr"
	"emeraldfox/internal/auth"
	"emeraldfox/internal/models"
	"emeraldfox/internal/repository"

	"github.com/google/uuid"
)

// UserRepository descreve o acesso aos dados exigido por UserService.
//
// A interface é declarada no consumidor para permitir a substituição do
// repositório em testes sem acoplar o serviço à implementação SQL.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// RegisterInput são os dados necessários para cadastrar um usuário.
type RegisterInput struct {
	Name  string
	Email string
}

// LoginInput são os dados necessários para autenticar um usuário.
type LoginInput struct {
	Email string
}

// Session é o resultado de um login bem-sucedido.
type Session struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

// UserService implementa as regras de negócio de usuários e autenticação.
type UserService struct {
	users  UserRepository
	tokens *auth.TokenManager
	log    *slog.Logger
}

// NewUserService cria o serviço de usuários.
func NewUserService(users UserRepository, tokens *auth.TokenManager, log *slog.Logger) *UserService {
	return &UserService{users: users, tokens: tokens, log: log}
}

// Register valida os dados informados e cadastra um novo usuário.
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return nil, err
	}

	email, err := validateEmail(input.Email)
	if err != nil {
		return nil, err
	}

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, apperr.Internalf(err, "verificar e-mail existente")
	}

	if exists {
		return nil, apperr.Conflict("Já existe um usuário cadastrado com este e-mail")
	}

	user := &models.User{
		ID:        uuid.New(),
		Name:      name,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.users.Create(ctx, user); err != nil {
		// Corrida entre a verificação acima e a inserção: o índice único do
		// banco é a garantia definitiva de unicidade.
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, apperr.Conflict("Já existe um usuário cadastrado com este e-mail")
		}
		return nil, apperr.Internalf(err, "criar usuário")
	}

	s.log.InfoContext(ctx, "Usuário cadastrado", slog.String("user_id", user.ID.String()))

	return user, nil
}

// Login autentica o usuário pelo e-mail e emite um token JWT.
//
// Como não há senha, a existência do e-mail é a própria credencial.
func (s *UserService) Login(ctx context.Context, input LoginInput) (*Session, error) {
	email, err := validateEmail(input.Email)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.log.WarnContext(ctx, "Tentativa de login com e-mail não cadastrado")
			return nil, apperr.NotFound("Nenhum usuário cadastrado com este e-mail")
		}
		return nil, apperr.Internalf(err, "buscar usuário por e-mail")
	}

	token, expiresAt, err := s.tokens.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, apperr.Internalf(err, "gerar token de acesso")
	}

	s.log.InfoContext(ctx, "Login realizado", slog.String("user_id", user.ID.String()))

	return &Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

// GetByID devolve o usuário correspondente ao identificador informado.
//
// É usado pelo middleware de autenticação para carregar o usuário do token.
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.NotFound("Usuário não encontrado")
		}
		return nil, apperr.Internalf(err, "buscar usuário por id")
	}

	return user, nil
}
