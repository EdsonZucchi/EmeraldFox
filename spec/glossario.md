# Glossário

Termos do domínio usados nas specs. Ao introduzir um termo novo, defina-o aqui
e use sempre o mesmo nome nas specs, no código e nas mensagens da API.

| Termo | Definição |
| ----- | --------- |
| Usuário | Pessoa cadastrada que se autentica na API. Todo recurso financeiro pertence a exatamente um usuário. |
| Usuário autenticado | Usuário identificado pelo token de acesso de uma requisição a uma rota protegida. |
| Token de acesso | JWT emitido no login que dá acesso às rotas protegidas até expirar. |
| Sessão | Resultado do login: token de acesso, instante de expiração e dados do usuário. Não é armazenada no servidor. |
| Rota pública | Rota acessível sem token (`/health`, `/register`, `/login`). |
| Rota protegida | Rota que exige token de acesso válido. Todo recurso financeiro fica em rotas protegidas. |
| Request ID | Identificador de uma requisição, devolvido em `X-Request-ID` e presente em todos os logs dela. |
| Normalização | Ajuste aplicado a um valor antes de validá-lo e gravá-lo (ex.: remover espaços, converter para minúsculas). |
