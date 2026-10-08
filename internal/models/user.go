// Package models contém as entidades de domínio da aplicação.
package models

import (
	"time"

	"github.com/google/uuid"
)

// User representa um usuário do sistema.
//
// A autenticação é feita apenas pelo e-mail, portanto a entidade não possui
// senha; novos campos de perfil podem ser adicionados sem impacto nas demais
// camadas.
type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
