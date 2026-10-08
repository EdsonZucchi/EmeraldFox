# EmeraldFox API

Base de uma REST API em Go para um sistema de organização financeira pessoal.

Esta primeira etapa entrega a fundação da aplicação — configuração, banco de
dados, usuários, autenticação por JWT, middlewares e logging estruturado —
preparada para receber os módulos financeiros (contas, categorias, lançamentos,
orçamentos) sem refatoração da estrutura.

## Stack

| Recurso            | Escolha                                        |
| ------------------ | ---------------------------------------------- |
| Linguagem          | Go 1.21+                                       |
| Roteamento         | `github.com/gorilla/mux`                       |
| Banco de dados     | PostgreSQL via `database/sql` + `github.com/lib/pq` |
| Autenticação       | `github.com/golang-jwt/jwt/v5` (HS256)         |
| Configuração       | `github.com/joho/godotenv`                     |
| Identificadores    | `github.com/google/uuid`                       |
| Logging            | `log/slog` (biblioteca padrão)                 |

Nenhum framework HTTP completo é utilizado: apenas a biblioteca padrão e o
`gorilla/mux`.

## Estrutura do projeto

```text
cmd/
    api/                     ponto de entrada: inicialização e shutdown
internal/
    appctx/                  valores por requisição no context.Context
    apperr/                  erro de aplicação (status HTTP + mensagem pública)
    auth/                    emissão e validação de JWT
    config/                  carga e validação do .env
    database/                conexão, pool e migrations
        migrations/          arquivos .sql embutidos no binário
    handlers/                camada HTTP (sem regra de negócio)
    middleware/              RequestID, Logging, Recovery, CORS e JWT
    models/                  entidades de domínio
    repository/              acesso a dados
    response/                envelope padronizado das respostas
    routes/                  declaração das rotas e composição dos middlewares
    service/                 regras de negócio
pkg/
    logger/                  logging estruturado com rotação diária
.env.example
README.md
```

O fluxo de uma requisição é sempre o mesmo:

```text
routes -> middleware -> handler -> service -> repository -> PostgreSQL
```

Cada camada só conhece a camada imediatamente abaixo, e a injeção de
dependências acontece em [cmd/api/main.go](cmd/api/main.go).

## Pré-requisitos

- Go 1.21 ou superior
- PostgreSQL 13 ou superior

## Configuração

1. Copie o arquivo de exemplo e ajuste os valores:

```bash
cp .env.example .env
```

2. Crie o banco de dados (o schema é criado automaticamente pelas migrations):

```bash
psql -U postgres -c "CREATE DATABASE finance;"
```

Variáveis disponíveis:

| Variável | Padrão | Descrição |
| -------- | ------ | --------- |
| `APP_PORT` | `8080` | Porta HTTP |
| `APP_ENV` | `development` | Ambiente da aplicação |
| `HTTP_READ_TIMEOUT` | `15s` | Timeout de leitura da requisição |
| `HTTP_WRITE_TIMEOUT` | `15s` | Timeout de escrita da resposta |
| `HTTP_IDLE_TIMEOUT` | `60s` | Timeout de conexões ociosas |
| `SHUTDOWN_TIMEOUT` | `15s` | Tempo máximo do encerramento controlado |
| `DB_HOST` | — | **Obrigatória** |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_NAME` | — | **Obrigatória** |
| `DB_USER` | — | **Obrigatória** |
| `DB_PASSWORD` | — | **Obrigatória** |
| `DB_SSLMODE` | `disable` | `disable`, `require`, `verify-full`, ... |
| `DB_MAX_OPEN_CONNS` | `25` | Tamanho máximo do pool |
| `DB_MAX_IDLE_CONNS` | `25` | Conexões ociosas mantidas |
| `DB_CONN_MAX_LIFETIME` | `5m` | Tempo de vida de cada conexão |
| `DB_CONN_MAX_IDLE_TIME` | `5m` | Tempo ocioso antes do descarte |
| `DB_AUTO_MIGRATE` | `true` | Aplica as migrations na inicialização |
| `JWT_SECRET` | — | **Obrigatória**, mínimo de 16 caracteres |
| `JWT_EXPIRATION` | `24h` | Validade do token |
| `JWT_ISSUER` | `emeraldfox-api` | Emissor registrado no token |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN` ou `ERROR` |
| `LOG_DIR` | `logs` | Diretório dos arquivos de log |
| `LOG_TO_STDOUT` | `true` | Espelha os logs no console |
| `CORS_ALLOWED_ORIGINS` | `*` | Lista separada por vírgulas |
| `CORS_ALLOWED_METHODS` | `GET,POST,PUT,PATCH,DELETE,OPTIONS` | Métodos permitidos |
| `CORS_ALLOWED_HEADERS` | `Authorization,Content-Type,X-Request-ID` | Cabeçalhos permitidos |
| `CORS_ALLOW_CREDENTIALS` | `false` | Permite envio de credenciais |
| `CORS_MAX_AGE` | `300` | Cache do preflight, em segundos |

A configuração é validada na inicialização: todos os problemas encontrados são
reportados de uma só vez e a aplicação não sobe com configuração inválida.
Variáveis já presentes no ambiente têm precedência sobre o arquivo `.env`, o que
permite sobrescrever valores em contêineres e pipelines de CI/CD.

## Executando

```bash
go mod download
go run ./cmd/api
```

Para gerar o executável:

```bash
go build -o finance-api ./cmd/api
```

```bash
./finance-api
```

A pasta `logs/` é criada automaticamente no diretório de execução.

## Migrations

Os arquivos `.sql` de [internal/database/migrations](internal/database/migrations)
são embutidos no binário e aplicados na inicialização quando
`DB_AUTO_MIGRATE=true`. Cada migration roda dentro de uma transação e é
registrada na tabela `schema_migrations`, de modo que nunca é executada duas
vezes.

Para criar uma nova migration, basta adicionar um arquivo seguindo a numeração:

```text
internal/database/migrations/0002_create_accounts_table.sql
```

## Endpoints

| Método | Rota | Autenticação | Descrição |
| ------ | ---- | ------------ | --------- |
| `GET`  | `/health` | Pública | Estado da API e do banco |
| `POST` | `/register` | Pública | Cadastra um usuário |
| `POST` | `/login` | Pública | Autentica por e-mail e devolve o JWT |
| `GET`  | `/me` | **Bearer token** | Dados do usuário autenticado |

### POST /register

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Edson Zucchi","email":"edson@exemplo.com"}'
```

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

### POST /login

O login é feito apenas pelo e-mail: não há senha. Se o e-mail existir, um token
é emitido; caso contrário a API responde `404`.

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"edson@exemplo.com"}'
```

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

### GET /me

```bash
curl http://localhost:8080/me \
  -H "Authorization: Bearer <token>"
```

O handler não valida o token e não consulta o banco: o usuário autenticado é
carregado pelo middleware e lido exclusivamente do `context.Context`.

## Padrão de respostas

Sucesso:

```json
{ "success": true, "data": {} }
```

Erro:

```json
{ "success": false, "message": "Descrição do erro" }
```

Status utilizados:

| Status | Situação |
| ------ | -------- |
| `400` | Corpo ausente, malformado ou com campos desconhecidos |
| `401` | Token ausente, inválido ou expirado |
| `404` | Recurso ou e-mail não encontrado |
| `409` | E-mail já cadastrado |
| `415` | `Content-Type` diferente de `application/json` |
| `422` | Dados válidos como JSON, porém inválidos como regra de negócio |
| `500` | Falha interna (detalhes apenas nos logs) |

Mensagens de erro nunca expõem detalhes internos — driver, SQL, stack trace ou
endereços de infraestrutura ficam restritos aos logs.

## Autenticação

O token é um JWT assinado em HS256 contendo:

| Claim | Conteúdo |
| ----- | -------- |
| `sub` | UUID do usuário |
| `iss` | Emissor (`JWT_ISSUER`) |
| `jti` | Identificador único do token |
| `iat` / `nbf` / `exp` | Emissão, validade inicial e expiração |
| `email` | E-mail do usuário |

O middleware [`middleware.Authenticate`](internal/middleware/auth.go):

1. extrai o token do cabeçalho `Authorization: Bearer <token>`;
2. valida assinatura, algoritmo, emissor e expiração;
3. extrai o ID do usuário da claim `sub`;
4. carrega o usuário no banco;
5. armazena o usuário no `context.Context` e adiciona `user_id` aos logs.

## Middlewares

Aplicados globalmente, do mais externo para o mais interno:

| Ordem | Middleware | Responsabilidade |
| ----- | ---------- | ---------------- |
| 1 | `RequestID` | Gera ou reaproveita o `X-Request-ID` e inicializa o contexto de logging |
| 2 | `Logging` | Registra uma linha por requisição com o status final |
| 3 | `Recovery` | Converte panics em `500` registrando o stack trace |
| 4 | `CORS` | Aplica a política de origens e responde ao preflight |

O middleware de autenticação é aplicado apenas ao subrouter das rotas
protegidas. Todos são independentes e reutilizáveis: a composição fica em
[internal/routes/routes.go](internal/routes/routes.go).

Como os middlewares globais envolvem o roteador (e não apenas as rotas
encontradas), respostas `404` e `405` também recebem `request_id`, log e CORS.

## Logging

- Biblioteca padrão `log/slog`, sem dependências externas.
- Formato **JSON Lines**: exatamente um objeto JSON por linha.
- Arquivos em `logs/`, criados automaticamente no diretório de execução.
- **Rotação diária**: um arquivo por dia, nomeado `YYYY-MM-DD.log`. A troca do
  arquivo acontece na primeira escrita do novo dia, sem reiniciar a aplicação.
- Nível mínimo configurável por `LOG_LEVEL` (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- Datas e horários em **UTC**, tanto no campo `timestamp` (RFC3339) quanto no
  nome do arquivo, garantindo consistência entre ambientes.

Exemplo de linha gerada pelo middleware de logging:

```json
{"timestamp":"2026-07-23T20:31:42Z","level":"INFO","message":"HTTP Request","method":"GET","path":"/me","status":200,"duration_ms":18.42,"ip":"192.168.0.10","user_agent":"curl/8.4.0","bytes":213,"request_id":"4c7c4e96d4f14cb0","user_id":"5c8df77e-56dc-4d44-a090-53dcb7f5f38e"}
```

Campos presentes em todos os registros: `timestamp`, `level` e `message`.
Quando aplicável, são incluídos `request_id`, `user_id`, `method`, `path`,
`status`, `duration_ms`, `ip`, `user_agent`, `bytes` e `error`.

O `request_id` é gerado automaticamente quando não enviado pelo cliente,
devolvido no cabeçalho `X-Request-ID` e propagado para todos os logs da
requisição. O `user_id` é adicionado assim que a autenticação identifica o
usuário — inclusive no log da própria requisição, emitido por um middleware mais
externo.

Erros internos (`5xx`) e panics são registrados com a causa original e o stack
trace completo, enquanto o cliente recebe apenas a mensagem genérica.

Por serem estruturados e consistentes, os arquivos podem ser ingeridos
diretamente por Grafana Loki, Elastic Stack, OpenSearch, Datadog ou Splunk.

## Testes

```bash
go test ./...
```

A suíte cobre a emissão e validação de tokens, as regras de negócio de usuários
e o fluxo HTTP completo (cadastro, login, rota protegida, CORS, `request_id`,
respostas de erro), usando um repositório em memória — não é necessário ter o
PostgreSQL disponível para executá-la.

Para testar a API em execução, a pasta [http](http/README.md) traz requisições
de exemplo de cada endpoint no formato do HTTP Client do GoLand/IntelliJ.

## Adicionando um novo módulo

A estrutura foi desenhada para que um módulo financeiro seja adicionado sem
alterar o que já existe. Para uma entidade `Account`, por exemplo:

1. `internal/database/migrations/0002_create_accounts_table.sql` — schema;
2. `internal/models/account.go` — entidade;
3. `internal/repository/account_repository.go` — consultas SQL;
4. `internal/service/account_service.go` — regras de negócio e a interface de
   repositório que o serviço consome;
5. `internal/handlers/account_handler.go` — leitura da requisição e resposta;
6. `internal/routes/routes.go` — registro das rotas no subrouter protegido;
7. `cmd/api/main.go` — ligação das dependências.

Os middlewares, o padrão de resposta, o tratamento de erros e o logging são
herdados automaticamente pelas novas rotas.
