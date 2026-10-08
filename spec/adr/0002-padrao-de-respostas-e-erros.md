# ADR-0002 — Padrão de respostas e erros

- **Status:** aceita
- **Data:** 2026-10-07 (registro de decisão já implementada)

## Contexto

Clientes da API precisam tratar sucesso e erro de forma uniforme em todas as
rotas, e falhas internas não podem vazar detalhes de infraestrutura.

## Decisão

- Envelope único para todas as respostas:
  - sucesso: `{"success": true, "data": ...}`;
  - erro: `{"success": false, "message": "..."}`.
- Toda mensagem de erro é escrita em português, é segura para exposição pública
  e é definida na spec do módulo.
- Mapeamento de status:

| Status | Uso |
| ------ | --- |
| `400` | Corpo ausente, malformado, com tipo errado ou com campos não previstos |
| `401` | Token ausente, inválido ou expirado |
| `403` | Autenticado, mas sem permissão (reservado para uso futuro) |
| `404` | Recurso inexistente — **inclusive recurso que pertence a outro usuário** |
| `405` | Método não suportado na rota |
| `409` | Conflito com o estado atual (ex.: duplicidade) |
| `413` | Corpo acima do limite |
| `415` | `Content-Type` não suportado |
| `422` | JSON válido, mas dados inválidos para a regra de negócio |
| `500` | Falha inesperada (mensagem genérica: "Erro interno do servidor") |
| `503` | Dependência indisponível (health check) |

- Falhas inesperadas são registradas no log com a causa original e o stack
  trace; o cliente recebe apenas a mensagem genérica.

## Consequências

- Specs de novos módulos só precisam listar as condições de erro, o status e a
  mensagem; o formato do corpo já está definido.
- Recurso de outro usuário responde `404` (e não `403`) para não revelar sua
  existência — regra que todos os módulos financeiros devem seguir.
