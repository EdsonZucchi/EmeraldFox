package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"emeraldfox/internal/auth"
	"emeraldfox/internal/config"
	"emeraldfox/internal/handlers"
	"emeraldfox/internal/models"
	"emeraldfox/internal/repository"
	"emeraldfox/internal/routes"
	"emeraldfox/internal/service"

	"github.com/google/uuid"
)

// fakeUserRepository substitui o PostgreSQL para exercitar a API de ponta a
// ponta (rotas, middlewares, handlers e serviços) sem infraestrutura externa.
type fakeUserRepository struct {
	users map[uuid.UUID]*models.User
}

func (f *fakeUserRepository) Create(_ context.Context, user *models.User) error {
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
	if _, err := f.GetByEmail(context.Background(), email); err != nil {
		return false, nil
	}

	return true, nil
}

// fakePinger simula a dependência de banco do health check.
type fakePinger struct{}

func (fakePinger) PingContext(context.Context) error { return nil }

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()

	api, _ := newTestAPIWithRepository(t)

	return api
}

// newTestAPIWithRepository monta a API e devolve também o repositório em
// memória, para testes que precisam manipular os dados diretamente.
func newTestAPIWithRepository(t *testing.T) (http.Handler, *fakeUserRepository) {
	t.Helper()

	repo := &fakeUserRepository{users: make(map[uuid.UUID]*models.User)}

	cfg := &config.Config{
		CORS: config.CORS{
			AllowedOrigins: []string{"https://app.exemplo.com"},
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
			MaxAge:         300,
		},
	}

	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tokens := auth.NewTokenManager("chave-de-teste-com-tamanho-suficiente", time.Hour, "emeraldfox-test")
	users := service.NewUserService(repo, tokens, log)

	api := routes.New(routes.Dependencies{
		Config: cfg,
		Logger: log,
		Tokens: tokens,
		Users:  users,
		Auth:   handlers.NewAuthHandler(users, log),
		User:   handlers.NewUserHandler(log),
		Health: handlers.NewHealthHandler(fakePinger{}, log),
	})

	return api, repo
}

// do executa uma requisição contra a API montada para os testes.
//
// Um cabeçalho informado com valor vazio é removido da requisição.
func do(t *testing.T, api http.Handler, method, path, body string, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		if value == "" {
			req.Header.Del(key)
			continue
		}
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	api.ServeHTTP(recorder, req)

	result := recorder.Result()
	defer func() {
		_ = result.Body.Close()
	}()

	payload := map[string]any{}

	raw, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("ler corpo da resposta: %v", err)
	}

	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("resposta não é um JSON válido (%s %s): %v — %s", method, path, err, raw)
		}
	}

	return result, payload
}

func TestFluxoCompletoDeCadastroLoginEMe(t *testing.T) {
	api := newTestAPI(t)

	// Cadastro.
	res, body := do(t, api, http.MethodPost, "/register", `{"name":"Edson Zucchi","email":"edson@exemplo.com"}`, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /register status = %d, esperado %d (%v)", res.StatusCode, http.StatusCreated, body)
	}

	if body["success"] != true {
		t.Errorf("POST /register success = %v, esperado true", body["success"])
	}

	created, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("POST /register não devolveu data: %v", body)
	}

	if created["email"] != "edson@exemplo.com" {
		t.Errorf("email cadastrado = %v", created["email"])
	}

	// Login.
	res, body = do(t, api, http.MethodPost, "/login", `{"email":"edson@exemplo.com"}`, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /login status = %d, esperado %d (%v)", res.StatusCode, http.StatusOK, body)
	}

	session, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("POST /login não devolveu data: %v", body)
	}

	token, ok := session["token"].(string)
	if !ok || token == "" {
		t.Fatalf("POST /login não devolveu token: %v", session)
	}

	// Rota protegida com o token emitido.
	res, body = do(t, api, http.MethodGet, "/me", "", map[string]string{"Authorization": "Bearer " + token})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /me status = %d, esperado %d (%v)", res.StatusCode, http.StatusOK, body)
	}

	me, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("GET /me não devolveu data: %v", body)
	}

	if me["id"] != created["id"] || me["email"] != created["email"] {
		t.Errorf("GET /me devolveu %v, esperado %v", me, created)
	}
}

func TestRegisterEmailDuplicadoRetornaConflito(t *testing.T) {
	api := newTestAPI(t)
	payload := `{"name":"Edson","email":"edson@exemplo.com"}`

	if res, body := do(t, api, http.MethodPost, "/register", payload, nil); res.StatusCode != http.StatusCreated {
		t.Fatalf("primeiro POST /register status = %d (%v)", res.StatusCode, body)
	}

	res, body := do(t, api, http.MethodPost, "/register", payload, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("segundo POST /register status = %d, esperado %d", res.StatusCode, http.StatusConflict)
	}

	if body["success"] != false || body["message"] == "" {
		t.Errorf("resposta de erro fora do padrão: %v", body)
	}
}

func TestLoginEmailNaoCadastrado(t *testing.T) {
	api := newTestAPI(t)

	res, body := do(t, api, http.MethodPost, "/login", `{"email":"ninguem@exemplo.com"}`, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("POST /login status = %d, esperado %d", res.StatusCode, http.StatusNotFound)
	}

	if body["success"] != false {
		t.Errorf("success = %v, esperado false", body["success"])
	}
}

func TestRotaProtegidaExigeTokenValido(t *testing.T) {
	api := newTestAPI(t)

	cases := map[string]map[string]string{
		"sem cabeçalho":     nil,
		"formato inválido":  {"Authorization": "Token abc"},
		"token inexistente": {"Authorization": "Bearer nao-e-um-jwt"},
	}

	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			res, body := do(t, api, http.MethodGet, "/me", "", headers)
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("GET /me status = %d, esperado %d", res.StatusCode, http.StatusUnauthorized)
			}

			if body["success"] != false {
				t.Errorf("success = %v, esperado false", body["success"])
			}

			if _, ok := body["data"]; ok {
				t.Errorf("resposta de erro não deve conter data: %v", body)
			}
		})
	}
}

func TestCorpoInvalidoRetornaBadRequest(t *testing.T) {
	api := newTestAPI(t)

	cases := map[string]string{
		"json malformado":  `{"email":`,
		"campo inesperado": `{"email":"edson@exemplo.com","senha":"123"}`,
		"tipo inválido":    `{"email":123}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			res, payload := do(t, api, http.MethodPost, "/login", body, nil)
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("POST /login status = %d, esperado %d (%v)", res.StatusCode, http.StatusBadRequest, payload)
			}
		})
	}
}

func TestRespostas401IncluemWWWAuthenticate(t *testing.T) {
	const challenge = `Bearer realm="emeraldfox"`

	api, repo := newTestAPIWithRepository(t)

	if res, body := do(t, api, http.MethodPost, "/register", `{"name":"Edson","email":"edson@exemplo.com"}`, nil); res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /register status = %d (%v)", res.StatusCode, body)
	}

	res, body := do(t, api, http.MethodPost, "/login", `{"email":"edson@exemplo.com"}`, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /login status = %d (%v)", res.StatusCode, body)
	}

	token := body["data"].(map[string]any)["token"].(string)

	t.Run("USR-RN-017 token ausente", func(t *testing.T) {
		res, _ := do(t, api, http.MethodGet, "/me", "", nil)
		if got := res.Header.Get("WWW-Authenticate"); got != challenge {
			t.Errorf("WWW-Authenticate = %q, esperado %q", got, challenge)
		}
	})

	t.Run("USR-RN-016 USR-RN-017 usuário do token removido", func(t *testing.T) {
		for id := range repo.users {
			delete(repo.users, id)
		}

		res, body := do(t, api, http.MethodGet, "/me", "", map[string]string{"Authorization": "Bearer " + token})
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("GET /me status = %d, esperado %d (%v)", res.StatusCode, http.StatusUnauthorized, body)
		}

		if body["message"] != "Token inválido" {
			t.Errorf("message = %v, esperado %q", body["message"], "Token inválido")
		}

		if got := res.Header.Get("WWW-Authenticate"); got != challenge {
			t.Errorf("WWW-Authenticate = %q, esperado %q", got, challenge)
		}
	})
}

func TestContentTypeObrigatorio(t *testing.T) {
	api := newTestAPI(t)
	payload := `{"email":"edson@exemplo.com"}`

	cases := map[string]string{
		"PLT-RN-006 cabeçalho ausente": "",
		"PLT-RN-006 outro tipo":        "text/plain",
	}

	for name, contentType := range cases {
		t.Run(name, func(t *testing.T) {
			res, body := do(t, api, http.MethodPost, "/login", payload, map[string]string{"Content-Type": contentType})
			if res.StatusCode != http.StatusUnsupportedMediaType {
				t.Fatalf("POST /login status = %d, esperado %d (%v)", res.StatusCode, http.StatusUnsupportedMediaType, body)
			}

			if body["message"] != "O corpo da requisição deve ser application/json" {
				t.Errorf("message = %v", body["message"])
			}
		})
	}

	t.Run("PLT-RN-006 parâmetros e maiúsculas aceitos", func(t *testing.T) {
		res, body := do(t, api, http.MethodPost, "/login", payload, map[string]string{"Content-Type": "Application/JSON; charset=utf-8"})
		if res.StatusCode == http.StatusUnsupportedMediaType {
			t.Fatalf("POST /login status = %d, Content-Type válido foi rejeitado (%v)", res.StatusCode, body)
		}
	})
}

func TestRequestIDEhGeradoOuPropagado(t *testing.T) {
	api := newTestAPI(t)

	res, _ := do(t, api, http.MethodGet, "/health", "", nil)
	if res.Header.Get("X-Request-ID") == "" {
		t.Error("X-Request-ID não foi gerado")
	}

	res, _ = do(t, api, http.MethodGet, "/health", "", map[string]string{"X-Request-ID": "req-do-cliente-123"})
	if got := res.Header.Get("X-Request-ID"); got != "req-do-cliente-123" {
		t.Errorf("X-Request-ID = %q, esperado o valor enviado pelo cliente", got)
	}

	// Identificadores fora do formato aceito são substituídos.
	res, _ = do(t, api, http.MethodGet, "/health", "", map[string]string{"X-Request-ID": "valor inválido\ncom quebra"})
	if got := res.Header.Get("X-Request-ID"); got == "valor inválido\ncom quebra" || got == "" {
		t.Errorf("X-Request-ID = %q, esperado um identificador gerado", got)
	}
}

func TestPreflightCORS(t *testing.T) {
	api := newTestAPI(t)

	res, _ := do(t, api, http.MethodOptions, "/me", "", map[string]string{
		"Origin":                        "https://app.exemplo.com",
		"Access-Control-Request-Method": "GET",
	})

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d, esperado %d", res.StatusCode, http.StatusNoContent)
	}

	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "https://app.exemplo.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}

	// Origem não autorizada não recebe o cabeçalho de permissão.
	res, _ = do(t, api, http.MethodOptions, "/me", "", map[string]string{
		"Origin":                        "https://site-malicioso.com",
		"Access-Control-Request-Method": "GET",
	})

	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origem não autorizada recebeu Access-Control-Allow-Origin = %q", got)
	}
}

func TestRotaInexistenteEMetodoNaoPermitido(t *testing.T) {
	api := newTestAPI(t)

	res, body := do(t, api, http.MethodGet, "/nao-existe", "", nil)
	if res.StatusCode != http.StatusNotFound || body["success"] != false {
		t.Errorf("GET /nao-existe status = %d, corpo = %v", res.StatusCode, body)
	}

	res, body = do(t, api, http.MethodDelete, "/me", "", nil)
	if res.StatusCode != http.StatusMethodNotAllowed || body["success"] != false {
		t.Errorf("DELETE /me status = %d, corpo = %v", res.StatusCode, body)
	}
}
