// Package auth concentra a emissão e a validação dos tokens JWT.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Erros devolvidos na validação de um token.
var (
	// ErrInvalidToken indica assinatura, formato ou emissor inválidos.
	ErrInvalidToken = errors.New("token inválido")
	// ErrExpiredToken indica um token com prazo de validade vencido.
	ErrExpiredToken = errors.New("token expirado")
)

// Claims são as informações transportadas pelo token.
//
// O identificador do usuário é registrado em "sub" (subject), conforme a
// RFC 7519.
type Claims struct {
	jwt.RegisteredClaims

	Email string `json:"email"`
}

// UserID devolve o identificador do usuário presente no token.
func (c *Claims) UserID() (uuid.UUID, error) {
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: subject não é um UUID válido", ErrInvalidToken)
	}

	return id, nil
}

// TokenManager emite e valida os tokens JWT da aplicação.
type TokenManager struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// NewTokenManager cria o gerenciador de tokens.
func NewTokenManager(secret string, expiration time.Duration, issuer string) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     issuer,
	}
}

// Expiration devolve o tempo de vida configurado para os tokens.
func (m *TokenManager) Expiration() time.Duration {
	return m.expiration
}

// GenerateToken emite um token assinado para o usuário informado e devolve
// também o instante de expiração.
func (m *TokenManager) GenerateToken(userID uuid.UUID, email string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.expiration)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Email: email,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("assinar token: %w", err)
	}

	return signed, expiresAt, nil
}

// ValidateToken verifica a assinatura, o emissor e a expiração do token,
// devolvendo as claims quando ele for válido.
func (m *TokenManager) ValidateToken(token string) (*Claims, error) {
	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(token, claims, m.keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err)
	}

	if !parsed.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// keyFunc devolve a chave de verificação garantindo o algoritmo esperado.
func (m *TokenManager) keyFunc(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
	}

	return m.secret, nil
}
