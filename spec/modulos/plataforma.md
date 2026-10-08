# Plataforma da API

| Campo | Valor |
| ----- | ----- |
| Prefixo | `PLT` |
| Status | implementada |
| Última atualização | 2026-10-07 |
| Depende de | [ADR-0002](../adr/0002-padrao-de-respostas-e-erros.md) |

## 1. Objetivo

Definir o comportamento comum a **todas** as rotas da API: formato das
respostas, leitura do corpo das requisições, rotas inexistentes, identificador
de requisição, CORS e verificação de saúde. Os módulos de negócio herdam essas
regras e não precisam repeti-las.

## 2. Entidades

Este módulo não possui entidades persistidas.

## 3. Regras de negócio

### Respostas

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-001` | Toda resposta de sucesso tem o corpo `{"success": true, "data": ...}`. Quando não há dados, `data` é um objeto vazio `{}`. | implementada |
| `PLT-RN-002` | Toda resposta de erro tem o corpo `{"success": false, "message": "..."}`, sem o campo `data`. | implementada |
| `PLT-RN-003` | Toda resposta usa `Content-Type: application/json; charset=utf-8` e `X-Content-Type-Options: nosniff`. | implementada |
| `PLT-RN-004` | Mensagens de erro nunca expõem detalhes internos (driver, SQL, stack trace, infraestrutura). Falhas inesperadas respondem `500` com a mensagem genérica; a causa real fica só nos logs. | implementada |
| `PLT-RN-005` | Datas e horários são sempre UTC e serializados em RFC 3339. | implementada |

### Corpo das requisições JSON

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-006` | Rotas que recebem JSON exigem o cabeçalho `Content-Type` com o tipo `application/json` (parâmetros como `charset` são aceitos; o tipo não diferencia maiúsculas de minúsculas). Cabeçalho ausente ou com outro tipo é rejeitado antes de o corpo ser lido. | implementada |
| `PLT-RN-007` | O corpo é limitado a 1 MiB. | implementada |
| `PLT-RN-008` | O corpo é obrigatório nas rotas que recebem JSON. | implementada |
| `PLT-RN-009` | Campos não previstos pela operação são rejeitados. | implementada |
| `PLT-RN-010` | O corpo deve conter exatamente um documento JSON. | implementada |

### Roteamento

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-011` | Rota inexistente responde `404` no padrão da API. | implementada |
| `PLT-RN-012` | Rota existente chamada com método não suportado responde `405` no padrão da API. | implementada |
| `PLT-RN-013` | Uma barra final na rota (`/me/`) é redirecionada para a rota sem barra (`/me`). | implementada |

### Identificador de requisição

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-014` | Toda resposta, inclusive `404`, `405` e erros, inclui o cabeçalho `X-Request-ID`. | implementada |
| `PLT-RN-015` | Um `X-Request-ID` enviado pelo cliente é reaproveitado se tiver de 1 a 64 caracteres, todos entre `A-Z`, `a-z`, `0-9`, `-` e `_`. Caso contrário, é gerado um identificador aleatório de 16 caracteres hexadecimais. | implementada |
| `PLT-RN-016` | O `request_id` é incluído em todos os logs da requisição. | implementada |

### CORS

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-017` | Origens, métodos, cabeçalhos, envio de credenciais e cache do preflight seguem a configuração `CORS_*`. Requisições sem `Origin` ou de origem não permitida não recebem cabeçalhos CORS. | implementada |
| `PLT-RN-018` | Com a origem curinga `*` e `CORS_ALLOW_CREDENTIALS=true`, a origem da requisição é devolvida no lugar do curinga. | implementada |
| `PLT-RN-019` | Preflight (`OPTIONS` com `Access-Control-Request-Method`) de origem permitida responde `204` sem chegar às rotas. | implementada |
| `PLT-RN-020` | O cabeçalho `X-Request-ID` é exposto ao navegador (`Access-Control-Expose-Headers`). | implementada |

### Saúde e robustez

| ID | Regra | Status |
| -- | ----- | ------ |
| `PLT-RN-021` | `GET /health` é pública e informa se a API e o banco de dados estão disponíveis. A verificação do banco tem limite de 2 segundos. | implementada |
| `PLT-RN-022` | Um panic durante a requisição nunca derruba o servidor: a resposta é `500` e o stack trace vai para o log. | implementada |
| `PLT-RN-023` | Cada requisição gera exatamente uma linha de log com método, rota, status final, duração, IP, user agent e bytes enviados. | implementada |

## 4. Operações

### Verificação de saúde — `GET /health`

- **Autenticação:** pública
- **Regras aplicadas:** `PLT-RN-021`

Saída (`200`):

```json
{ "success": true, "data": { "status": "ok", "database": "up" } }
```

Erros:

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| Banco indisponível ou sem resposta em 2 s | `503` | Serviço temporariamente indisponível |

### Erros comuns a todas as rotas

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| `Content-Type` ausente ou diferente de `application/json` | `415` | O corpo da requisição deve ser application/json |
| Corpo acima de 1 MiB | `413` | O corpo da requisição excede o tamanho máximo permitido |
| Corpo vazio | `400` | O corpo da requisição é obrigatório |
| JSON malformado ou incompleto | `400` | O corpo da requisição não é um JSON válido |
| Campo com tipo errado | `400` | O campo "<campo>" possui um tipo inválido |
| Campo não previsto | `400` | O campo "<campo>" não é aceito nesta requisição |
| Mais de um documento JSON no corpo | `400` | O corpo da requisição deve conter um único objeto JSON |
| Outra falha de leitura do corpo | `400` | Não foi possível ler o corpo da requisição |
| Rota inexistente | `404` | Recurso não encontrado |
| Método não suportado | `405` | Método não permitido para este recurso |
| Falha inesperada | `500` | Erro interno do servidor |

## 5. Critérios de aceite

- **`PLT-CA-001`** — Quando envio `{"email":`, `{"email":123}` ou um campo não
  previsto para `POST /login`, então recebo `400`.
- **`PLT-CA-002`** — Quando faço uma requisição sem `X-Request-ID`, então a
  resposta traz um identificador gerado; quando envio um identificador válido,
  então ele é devolvido igual.
- **`PLT-CA-003`** — Dado uma origem permitida, quando envio um preflight, então
  recebo `204` com os cabeçalhos CORS configurados.
- **`PLT-CA-004`** — Quando chamo uma rota inexistente, então recebo `404`;
  quando chamo uma rota existente com método não suportado, então recebo `405`;
  ambas no padrão da API e com `X-Request-ID`.

## 6. Fora de escopo

- Versionamento da API (prefixo `/v1`).
- Paginação, ordenação e filtros: serão definidos em ADR quando o primeiro
  módulo com listagem for especificado.
- Limite de requisições por cliente (rate limiting).
- Compressão de respostas.

## 7. Questões em aberto

Nenhuma.

### Decisões tomadas

| Data | Questão | Decisão |
| ---- | ------- | ------- |
| 2026-10-07 | Requisição com corpo e sem `Content-Type` era aceita. | Exigir o cabeçalho; ausência responde `415` (`PLT-RN-006`). |
| 2026-10-07 | A tabela de status do `README.md` não listava `405`, `413` e `503`. | Atualizar o README conforme a [ADR-0002](../adr/0002-padrao-de-respostas-e-erros.md). |

### Regras sem teste automatizado dedicado

- `PLT-RN-003`, `PLT-RN-007` (`413`), `PLT-RN-008`,
  `PLT-RN-010`, `PLT-RN-013`, `PLT-RN-015` (identificador inválido descartado),
  `PLT-RN-018`, `PLT-RN-021`, `PLT-RN-022`, `PLT-RN-023`

## 8. Histórico de alterações

| Data | Alteração |
| ---- | --------- |
| 2026-10-07 | Criação da spec a partir do comportamento já implementado (engenharia reversa do código). |
| 2026-10-07 | `PLT-RN-006` alterada: `Content-Type: application/json` passa a ser obrigatório nas rotas que recebem JSON. |
| 2026-10-07 | `PLT-RN-006` implementada e coberta por testes; README atualizado com os status `405`, `413` e `503`. |
