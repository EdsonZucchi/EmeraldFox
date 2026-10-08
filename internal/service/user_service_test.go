package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"emeraldfox/internal/apperr"
	"emeraldfox/internal/auth"
	"emeraldfox/internal/models"
	"emeraldfox/internal/repository"
	"emeraldfox/internal/service"

	"github.com/google/uuid"
)

// fakeUserRepository é uma implementação em memória de service.UserRepository,
// suficiente para exercitar as regras de negócio sem banco de dados.
type fakeUserRepository struct {
	users     map[uuid.UUID]*models.User
	createErr error
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{users: make(map[uuid.UUID]*models.User)}
}

func (f *fakeUserRepository) Create(_ context.Context, user *models.User) error {
	if f.createErr != nil {
		return f.createErr
	}

	copied := *user
	f.users[user.ID] = &copied

	return nil
}

func (f *fakeUserRepository) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if user, ok := f.users[id]; ok {
		return user, nil
	}

	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, err := f.GetByEmail(context.Background(), email)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func newTestService(repo service.UserRepository) *service.UserService {
	tokens := auth.NewTokenManager("chave-de-teste-com-tamanho-suficiente", time.Hour, "emeraldfox-test")
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	return service.NewUserService(repo, tokens, log)
}

// statusOf extrai o status HTTP associado ao erro de aplicação.
func statusOf(t *testing.T, err error) int {
	t.Helper()

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("erro %v não é um *apperr.Error", err)
	}

	return appErr.Status
}

func TestRegisterNormalizaDados(t *testing.T) {
	svc := newTestService(newFakeUserRepository())

	user, err := svc.Register(context.Background(), service.RegisterInput{
		Name:  "  Edson   Zucchi ",
		Email: "  Edson@Exemplo.COM ",
	})
	if err != nil {
		t.Fatalf("Register devolveu erro: %v", err)
	}

	if user.Name != "Edson Zucchi" {
		t.Errorf("nome = %q, esperado %q", user.Name, "Edson Zucchi")
	}

	if user.Email != "edson@exemplo.com" {
		t.Errorf("email = %q, esperado %q", user.Email, "edson@exemplo.com")
	}

	if user.ID == uuid.Nil {
		t.Error("id do usuário não foi gerado")
	}

	if user.CreatedAt.IsZero() {
		t.Error("created_at não foi preenchido")
	}
}

func TestRegisterEntradasInvalidas(t *testing.T) {
	svc := newTestService(newFakeUserRepository())

	cases := map[string]service.RegisterInput{
		"nome vazio":     {Name: "   ", Email: "valido@exemplo.com"},
		"nome curto":     {Name: "A", Email: "valido@exemplo.com"},
		"nome longo":     {Name: strings.Repeat("a", 121), Email: "valido@exemplo.com"},
		"email vazio":    {Name: "Usuário", Email: ""},
		"email inválido": {Name: "Usuário", Email: "sem-arroba"},
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), input)
			if err == nil {
				t.Fatal("esperado erro de validação")
			}

			if status := statusOf(t, err); status != http.StatusUnprocessableEntity {
				t.Errorf("status = %d, esperado %d", status, http.StatusUnprocessableEntity)
			}
		})
	}
}

func TestRegisterEmailDuplicado(t *testing.T) {
	svc := newTestService(newFakeUserRepository())
	input := service.RegisterInput{Name: "Edson", Email: "edson@exemplo.com"}

	if _, err := svc.Register(context.Background(), input); err != nil {
		t.Fatalf("primeiro Register devolveu erro: %v", err)
	}

	// O mesmo e-mail em outro formato deve ser recusado após a normalização.
	input.Email = "EDSON@exemplo.com"

	_, err := svc.Register(context.Background(), input)
	if err == nil {
		t.Fatal("esperado erro de conflito")
	}

	if status := statusOf(t, err); status != http.StatusConflict {
		t.Errorf("status = %d, esperado %d", status, http.StatusConflict)
	}
}

func TestRegisterFalhaDoRepositorioNaoVazaDetalhes(t *testing.T) {
	repo := newFakeUserRepository()
	repo.createErr = errors.New("connection refused: 10.0.0.5:5432")

	_, err := newTestService(repo).Register(context.Background(), service.RegisterInput{
		Name:  "Edson",
		Email: "edson@exemplo.com",
	})
	if err == nil {
		t.Fatal("esperado erro interno")
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("erro %v não é um *apperr.Error", err)
	}

	if appErr.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, esperado %d", appErr.Status, http.StatusInternalServerError)
	}

	if strings.Contains(appErr.Message, "10.0.0.5") {
		t.Errorf("mensagem pública vazou detalhes internos: %q", appErr.Message)
	}

	if appErr.Stack == "" {
		t.Error("erro interno deveria conter stack trace para os logs")
	}
}

func TestLoginGeraTokenValido(t *testing.T) {
	repo := newFakeUserRepository()
	svc := newTestService(repo)
	ctx := context.Background()

	created, err := svc.Register(ctx, service.RegisterInput{Name: "Edson", Email: "edson@exemplo.com"})
	if err != nil {
		t.Fatalf("Register devolveu erro: %v", err)
	}

	session, err := svc.Login(ctx, service.LoginInput{Email: "EDSON@EXEMPLO.COM"})
	if err != nil {
		t.Fatalf("Login devolveu erro: %v", err)
	}

	if session.Token == "" {
		t.Fatal("token não foi gerado")
	}

	if session.User.ID != created.ID {
		t.Errorf("usuário da sessão = %s, esperado %s", session.User.ID, created.ID)
	}

	if !session.ExpiresAt.After(time.Now()) {
		t.Errorf("expiração deveria ser futura, recebido %s", session.ExpiresAt)
	}
}

func TestLoginEmailNaoCadastrado(t *testing.T) {
	svc := newTestService(newFakeUserRepository())

	_, err := svc.Login(context.Background(), service.LoginInput{Email: "ninguem@exemplo.com"})
	if err == nil {
		t.Fatal("esperado erro para e-mail não cadastrado")
	}

	if status := statusOf(t, err); status != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", status, http.StatusNotFound)
	}
}

func TestGetByIDInexistente(t *testing.T) {
	svc := newTestService(newFakeUserRepository())

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("esperado erro para usuário inexistente")
	}

	if status := statusOf(t, err); status != http.StatusNotFound {
		t.Errorf("status = %d, esperado %d", status, http.StatusNotFound)
	}
}
