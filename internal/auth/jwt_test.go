package auth_test

import (
	"errors"
	"testing"
	"time"

	"emeraldfox/internal/auth"

	"github.com/google/uuid"
)

const (
	testSecret = "chave-de-teste-com-tamanho-suficiente"
	testIssuer = "emeraldfox-test"
)

func TestGenerateAndValidateToken(t *testing.T) {
	manager := auth.NewTokenManager(testSecret, time.Hour, testIssuer)
	userID := uuid.New()

	token, expiresAt, err := manager.GenerateToken(userID, "usuario@exemplo.com")
	if err != nil {
		t.Fatalf("GenerateToken devolveu erro: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken devolveu token vazio")
	}

	if time.Until(expiresAt) <= 0 {
		t.Fatalf("expiração deveria ser futura, recebido %s", expiresAt)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken devolveu erro: %v", err)
	}

	got, err := claims.UserID()
	if err != nil {
		t.Fatalf("UserID devolveu erro: %v", err)
	}

	if got != userID {
		t.Errorf("user id = %s, esperado %s", got, userID)
	}

	if claims.Email != "usuario@exemplo.com" {
		t.Errorf("email = %q, esperado %q", claims.Email, "usuario@exemplo.com")
	}
}

func TestValidateTokenExpirado(t *testing.T) {
	manager := auth.NewTokenManager(testSecret, -time.Minute, testIssuer)

	token, _, err := manager.GenerateToken(uuid.New(), "usuario@exemplo.com")
	if err != nil {
		t.Fatalf("GenerateToken devolveu erro: %v", err)
	}

	if _, err := manager.ValidateToken(token); !errors.Is(err, auth.ErrExpiredToken) {
		t.Errorf("erro = %v, esperado ErrExpiredToken", err)
	}
}

func TestValidateTokenAssinadoComOutraChave(t *testing.T) {
	issuer := auth.NewTokenManager(testSecret, time.Hour, testIssuer)
	validator := auth.NewTokenManager("outra-chave-completamente-diferente", time.Hour, testIssuer)

	token, _, err := issuer.GenerateToken(uuid.New(), "usuario@exemplo.com")
	if err != nil {
		t.Fatalf("GenerateToken devolveu erro: %v", err)
	}

	if _, err := validator.ValidateToken(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("erro = %v, esperado ErrInvalidToken", err)
	}
}

func TestValidateTokenDeOutroEmissor(t *testing.T) {
	issuer := auth.NewTokenManager(testSecret, time.Hour, "outra-api")
	validator := auth.NewTokenManager(testSecret, time.Hour, testIssuer)

	token, _, err := issuer.GenerateToken(uuid.New(), "usuario@exemplo.com")
	if err != nil {
		t.Fatalf("GenerateToken devolveu erro: %v", err)
	}

	if _, err := validator.ValidateToken(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("erro = %v, esperado ErrInvalidToken", err)
	}
}

func TestValidateTokenMalformado(t *testing.T) {
	manager := auth.NewTokenManager(testSecret, time.Hour, testIssuer)

	if _, err := manager.ValidateToken("isto-nao-e-um-jwt"); !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("erro = %v, esperado ErrInvalidToken", err)
	}
}
