# Requisições HTTP

Requisições para testar a API manualmente, no formato do HTTP Client do
GoLand/IntelliJ. Cada pasta corresponde a um módulo e cada arquivo `.http` a um
endpoint:

```text
http/
    http-client.env.json   variáveis por ambiente (baseUrl, name, email)
    health/health.http     GET  /health
    auth/register.http     POST /register
    auth/login.http        POST /login
    user/me.http           GET  /me
```

## Como usar

1. Suba a API (`go run ./cmd/api`).
2. Abra qualquer arquivo `.http` e selecione o ambiente `dev` no topo do editor.
3. Na primeira vez, execute nesta ordem:
   1. o primeiro request de `auth/register.http`, que cria o usuário do ambiente;
   2. o primeiro request de `auth/login.http`, que salva o JWT em `{{token}}`;
   3. os requests de `user/me.http`, que usam o token salvo.

Cada request tem um bloco `> {% ... %}` com asserções sobre o status e o corpo da
resposta. O resultado aparece na aba de testes do HTTP Client.

## Novos módulos

Crie uma pasta por módulo (ex.: `accounts/`) com um arquivo por endpoint
(`list.http`, `create.http`, ...). Rotas protegidas usam o cabeçalho
`Authorization: Bearer {{token}}`.
