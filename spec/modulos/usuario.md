# Usuário e autenticação

| Campo | Valor |
| ----- | ----- |
| Prefixo | `USR` |
| Status | implementada |
| Última atualização | 2026-10-07 |
| Depende de | [plataforma.md](plataforma.md), [ADR-0001](../adr/0001-autenticacao-por-email-com-jwt.md), [ADR-0002](../adr/0002-padrao-de-respostas-e-erros.md) |

## 1. Objetivo

Permitir que uma pessoa se cadastre na API, obtenha um token de acesso e use
esse token para acessar as rotas protegidas. O usuário é o dono de todos os
recursos financeiros que serão criados nos próximos módulos.

O EmeraldFox é de **uso local**. O usuário existe principalmente para
relacionar os dados financeiros a um dono e permitir mais de um usuário no
futuro; não é um mecanismo de segurança (ver
[ADR-0001](../adr/0001-autenticacao-por-email-com-jwt.md)).

## 2. Entidades

### Usuário

| Campo | Tipo | Obrigatório | Regras |
| ----- | ---- | ----------- | ------ |
| `id` | UUID | sim | Gerado pelo sistema no cadastro (UUID v4). Nunca informado pelo cliente. |
| `name` | texto | sim | Normalizado e validado conforme `USR-RN-001` a `USR-RN-003`. |
| `email` | texto | sim | Normalizado e validado conforme `USR-RN-004` a `USR-RN-007`. Único em todo o sistema. |
| `created_at` | data/hora UTC | sim | Instante do cadastro, gerado pelo sistema. Serializado em RFC 3339. |

O usuário **não possui senha** (ver [ADR-0001](../adr/0001-autenticacao-por-email-com-jwt.md)).

### Token de acesso

JWT assinado com HS256, emitido no login. Não é persistido.

| Claim | Conteúdo |
| ----- | -------- |
| `sub` | `id` do usuário |
| `email` | `email` do usuário no momento da emissão |
| `iss` | Emissor configurado em `JWT_ISSUER` (padrão `emeraldfox-api`) |
| `jti` | Identificador único do token (UUID) |
| `iat` / `nbf` | Instante da emissão |
| `exp` | Instante da emissão + `JWT_EXPIRATION` (padrão `24h`) |

## 3. Regras de negócio

### Nome

| ID | Regra | Status |
| -- | ----- | ------ |
| `USR-RN-001` | O nome é normalizado antes da validação: espaços nas extremidades são removidos e sequências de espaços internos viram um único espaço (`"  Edson   Zucchi "` → `"Edson Zucchi"`). | implementada |
| `USR-RN-002` | O nome é obrigatório. Vazio ou composto apenas de espaços é rejeitado. | implementada |
| `USR-RN-003` | O nome normalizado deve ter entre 2 e 120 caracteres (contados como caracteres Unicode, não bytes). | implementada |

### E-mail

| ID | Regra | Status |
| -- | ----- | ------ |
| `USR-RN-004` | O e-mail é normalizado antes da validação: espaços nas extremidades são removidos e todo o texto é convertido para minúsculas. A mesma normalização vale no cadastro e no login. | implementada |
| `USR-RN-005` | O e-mail é obrigatório. | implementada |
| `USR-RN-006` | O e-mail normalizado deve ter no máximo 255 bytes. | implementada |
| `USR-RN-007` | O e-mail deve ser um endereço simples válido (`usuario@dominio`). Formatos com nome de exibição (`Edson <edson@exemplo.com>`) são rejeitados. | implementada |
| `USR-RN-008` | O e-mail é único no sistema, comparado após a normalização: `Edson@Exemplo.com` e `edson@exemplo.com` são o mesmo e-mail. A unicidade também é garantida pelo banco, mesmo com cadastros simultâneos. | implementada |

### Cadastro e login

| ID | Regra | Status |
| -- | ----- | ------ |
| `USR-RN-009` | No cadastro, o nome é validado antes do e-mail; o primeiro erro encontrado é o único devolvido. | implementada |
| `USR-RN-010` | O login é feito apenas pelo e-mail. Se existir um usuário com o e-mail normalizado, um token de acesso é emitido. | implementada |
| `USR-RN-011` | Login com e-mail não cadastrado responde `404` e não emite token. | implementada |
| `USR-RN-012` | A resposta do login contém o token, o instante de expiração (`expires_at`, UTC) e os dados do usuário. | implementada |

### Acesso às rotas protegidas

| ID | Regra | Status |
| -- | ----- | ------ |
| `USR-RN-013` | Rotas protegidas exigem o cabeçalho `Authorization: Bearer <token>`. O prefixo `Bearer` não diferencia maiúsculas de minúsculas. | implementada |
| `USR-RN-014` | O token só é aceito se: for assinado com HS256 pela chave da aplicação, tiver o emissor configurado, possuir `exp` e não estiver expirado. | implementada |
| `USR-RN-015` | A claim `sub` deve ser um UUID válido; caso contrário o token é inválido. | implementada |
| `USR-RN-016` | O usuário do token é recarregado do banco a cada requisição. Se ele não existir mais, a requisição é recusada como token inválido, sem revelar que o cadastro deixou de existir. | implementada |
| `USR-RN-017` | Toda resposta `401` de rota protegida — token ausente, inválido, expirado ou de usuário que não existe mais — inclui o cabeçalho `WWW-Authenticate: Bearer realm="emeraldfox"`. | implementada |
| `USR-RN-018` | `GET /me` devolve os dados do usuário autenticado sem nova consulta ao banco, usando o usuário já carregado na autenticação. | implementada |

### Segurança e observabilidade

| ID | Regra | Status |
| -- | ----- | ------ |
| `USR-RN-019` | O identificador do usuário autenticado (`user_id`) é adicionado a todos os logs da requisição, inclusive à linha de log da própria requisição. | implementada |
| `USR-RN-020` | Cadastro, login bem-sucedido e tentativa de login com e-mail inexistente são registrados no log; o e-mail **não** é gravado no log. | implementada |

## 4. Operações

### Cadastrar usuário — `POST /register`

- **Autenticação:** pública
- **Regras aplicadas:** `USR-RN-001` a `USR-RN-009`, mais as regras de corpo de
  requisição de [plataforma.md](plataforma.md)

Entrada (somente estes campos são aceitos):

```json
{ "name": "Edson Zucchi", "email": "edson@exemplo.com" }
```

Saída (`201`):

```json
{
  "success": true,
  "data": {
    "id": "5c8df77e-56dc-4d44-a090-53dcb7f5f38e",
    "name": "Edson Zucchi",
    "email": "edson@exemplo.com",
    "created_at": "2026-07-23T20:31:42.512Z"
  }
}
```

Erros:

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| Nome vazio após normalização | `422` | O nome é obrigatório |
| Nome com menos de 2 caracteres | `422` | O nome deve ter no mínimo 2 caracteres |
| Nome com mais de 120 caracteres | `422` | O nome deve ter no máximo 120 caracteres |
| E-mail vazio após normalização | `422` | O e-mail é obrigatório |
| E-mail com mais de 255 bytes | `422` | O e-mail deve ter no máximo 255 caracteres |
| E-mail em formato inválido | `422` | O e-mail informado é inválido |
| E-mail já cadastrado | `409` | Já existe um usuário cadastrado com este e-mail |
| Corpo inválido | `400`/`413`/`415` | Ver [plataforma.md](plataforma.md) |

### Login — `POST /login`

- **Autenticação:** pública
- **Regras aplicadas:** `USR-RN-004` a `USR-RN-007`, `USR-RN-010` a `USR-RN-012`

Entrada (somente este campo é aceito):

```json
{ "email": "edson@exemplo.com" }
```

Saída (`200`):

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-07-24T20:31:42Z",
    "user": {
      "id": "5c8df77e-56dc-4d44-a090-53dcb7f5f38e",
      "name": "Edson Zucchi",
      "email": "edson@exemplo.com",
      "created_at": "2026-07-23T20:31:42.512Z"
    }
  }
}
```

Erros:

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| E-mail vazio, longo ou inválido | `422` | Mesmas mensagens do cadastro |
| E-mail não cadastrado | `404` | Nenhum usuário cadastrado com este e-mail |
| Corpo inválido | `400`/`413`/`415` | Ver [plataforma.md](plataforma.md) |

### Usuário autenticado — `GET /me`

- **Autenticação:** Bearer token
- **Regras aplicadas:** `USR-RN-013` a `USR-RN-018`

Saída (`200`): o objeto do usuário, no mesmo formato de `data` do cadastro.

Erros (valem para **todas** as rotas protegidas):

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| Cabeçalho ausente, fora do formato `Bearer <token>` ou com token vazio | `401` | Token de autenticação não informado |
| Token expirado | `401` | Token expirado |
| Assinatura, algoritmo, emissor ou `sub` inválidos; token malformado | `401` | Token inválido |
| Token válido, mas usuário não existe mais | `401` | Token inválido |

## 5. Critérios de aceite

- **`USR-CA-001`** — Dado um e-mail não cadastrado, quando eu me cadastro com
  `"  Edson   Zucchi "` e `"  Edson@Exemplo.COM "`, então recebo `201` com
  nome `Edson Zucchi`, e-mail `edson@exemplo.com`, `id` e `created_at`
  preenchidos.
- **`USR-CA-002`** — Dado um usuário cadastrado com `edson@exemplo.com`, quando
  tento cadastrar `EDSON@exemplo.com`, então recebo `409`.
- **`USR-CA-003`** — Dado um usuário cadastrado, quando faço login com o seu
  e-mail, então recebo `200` com um token válido, `expires_at` e os dados do
  usuário.
- **`USR-CA-004`** — Dado um e-mail não cadastrado, quando faço login, então
  recebo `404` e nenhum token.
- **`USR-CA-005`** — Dado um token emitido no login, quando chamo `GET /me` com
  `Authorization: Bearer <token>`, então recebo `200` com os dados do usuário
  dono do token.
- **`USR-CA-006`** — Quando chamo `GET /me` sem cabeçalho, com formato
  diferente de `Bearer` ou com um texto que não é JWT, então recebo `401` com
  `success: false` e sem `data`.
- **`USR-CA-007`** — Dado um token expirado, assinado com outra chave ou de
  outro emissor, quando chamo uma rota protegida, então recebo `401`.

## 6. Fora de escopo

- Senha, confirmação de e-mail, recuperação de acesso e autenticação em dois
  fatores.
- Edição de nome ou e-mail e exclusão de conta.
- Revogação de token (logout) e refresh token: o token vale até expirar.
- Papéis e permissões (todo usuário tem o mesmo nível de acesso).
- Limite de tentativas de login (rate limiting).

## 7. Questões em aberto

1. **Limite do e-mail em bytes.** O limite de 255 é contado em bytes, mas a
   mensagem fala em "caracteres". Para e-mails só com ASCII não há diferença.
   Manter assim? (afeta `USR-RN-006`)

### Decisões tomadas

| Data | Questão | Decisão |
| ---- | ------- | ------- |
| 2026-10-07 | Login sem senha permite entrar na conta de quem se conhece o e-mail. | Aceito. Uso local; o usuário serve para relacionar tabelas. Revisar se a API for exposta em rede ([ADR-0001](../adr/0001-autenticacao-por-email-com-jwt.md)). |
| 2026-10-07 | `404` no login e `409` no cadastro revelam quais e-mails estão cadastrados. | Aceito pelo mesmo motivo; respostas mantidas. |
| 2026-10-07 | `401` de usuário removido saía sem `WWW-Authenticate`. | Padronizar: todo `401` de rota protegida inclui o cabeçalho (`USR-RN-017`). |

### Regras sem teste automatizado dedicado

Estas regras estão implementadas, mas ainda não têm teste que as comprove.
O `desenvolvedor` pode cobri-las sem alterar comportamento:

- `USR-RN-006` (e-mail acima de 255 bytes)
- `USR-RN-007` (rejeição de `Nome <email>`)
- `USR-RN-009` (ordem de validação)
- `USR-RN-013` (prefixo `bearer` em minúsculas aceito)
- `USR-RN-015` (`sub` que não é UUID)
- `USR-RN-019` e `USR-RN-020` (conteúdo dos logs)

## 8. Histórico de alterações

| Data | Alteração |
| ---- | --------- |
| 2026-10-07 | Criação da spec a partir do comportamento já implementado (engenharia reversa do código). |
| 2026-10-07 | Registradas as decisões sobre login sem senha e descoberta de e-mails (aceitos para uso local). `USR-RN-017` alterada: o `401` de usuário removido passa a incluir `WWW-Authenticate`. |
| 2026-10-07 | `USR-RN-017` implementada; `USR-RN-016` e `USR-RN-017` passam a ter testes. |
