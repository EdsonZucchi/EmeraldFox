# ADR-0001 — Autenticação por e-mail com JWT

- **Status:** aceita
- **Data:** 2026-10-07 (registro de decisão já implementada)

## Contexto

O EmeraldFox é de uso local. O usuário existe para relacionar os dados
financeiros a um dono e permitir, no futuro, mais de uma pessoa usando a mesma
instalação — não como barreira de segurança. Por isso a identificação deve ter
o mínimo de atrito possível.

## Decisão

- O usuário se identifica **apenas pelo e-mail**; não há senha nem outro fator.
- Após o login, a API emite um **JWT HS256** com `sub` (id do usuário), `email`,
  `iss`, `jti`, `iat`, `nbf` e `exp`. A chave é `JWT_SECRET` (mínimo de 16
  caracteres), a validade é `JWT_EXPIRATION` (padrão `24h`) e o emissor é
  `JWT_ISSUER`.
- O token não é armazenado (stateless), mas o usuário é recarregado do banco em
  toda requisição autenticada, de modo que um usuário removido perde o acesso
  imediatamente.
- Apenas o algoritmo HS256 é aceito na validação, e `exp` é obrigatório.

## Consequências

- Implementação simples e sem estado no servidor; escala horizontalmente
  compartilhando apenas `JWT_SECRET`.
- Não há logout nem revogação: um token vazado vale até expirar. Trocar
  `JWT_SECRET` invalida todos os tokens de uma vez.
- **Sem barreira de segurança:** quem conhece o e-mail de um usuário obtém
  acesso à conta dele, e as respostas de login e cadastro revelam quais
  e-mails existem. Aceito conscientemente por ser uso local (decisão de
  2026-10-07).
- **Gatilho de revisão:** se a API passar a ser exposta em rede (outra máquina,
  servidor, internet), esta ADR deve ser substituída por uma que introduza um
  fator de autenticação (senha, link mágico ou código de uso único).
