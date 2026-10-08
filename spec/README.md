# Especificação do EmeraldFox

Esta pasta é a **fonte única da verdade** sobre as regras de negócio do
EmeraldFox. O código implementa o que está aqui; se o código e a spec
divergirem, a spec define o comportamento esperado.

- Quem escreve: o agente **arquiteto** (`.claude/agents/arquiteto.md`), a partir
  das decisões do usuário.
- Quem lê e implementa: o agente **desenvolvedor**
  (`.claude/agents/desenvolvedor.md`), que nunca altera esta pasta.

## Estrutura

```text
spec/
    README.md          este arquivo: convenções e índice
    glossario.md       termos do domínio usados em todas as specs
    _template.md       modelo para a spec de um módulo
    modulos/           uma spec por módulo de negócio (conta.md, categoria.md, ...)
    adr/               decisões de arquitetura transversais (NNNN-titulo.md)
```

## Fluxo de trabalho

```text
usuário descreve a necessidade
  -> arquiteto escreve/atualiza a spec (status: rascunho) e levanta questões
  -> usuário responde e aprova (status: aprovada)
  -> desenvolvedor implementa as regras aprovadas, com testes
  -> desenvolvedor reporta os IDs implementados e pendências
  -> arquiteto marca as regras como implementada e resolve as pendências
```

## Status

| Status | Significado |
| ------ | ----------- |
| `rascunho` | Em definição. Pode mudar; **não** deve ser implementada. |
| `aprovada` | Confirmada pelo usuário e pronta para implementação. |
| `implementada` | Implementada e coberta por testes. |
| `removida` | Deixou de valer. O ID é mantido e nunca reaproveitado. |

Uma regra `implementada` que for alterada volta para `rascunho` (ou `aprovada`)
e a mudança é descrita no *Histórico de alterações* da spec.

## Identificadores

| Tipo | Formato | Exemplo |
| ---- | ------- | ------- |
| Regra de negócio | `<PREFIXO>-RN-NNN` | `CONTA-RN-001` |
| Critério de aceite | `<PREFIXO>-CA-NNN` | `CONTA-CA-001` |
| Decisão de arquitetura | `ADR-NNNN` | `ADR-0001` |

O prefixo é definido na spec de cada módulo e registrado no índice abaixo. Os
IDs são citados nos testes do código, garantindo rastreabilidade entre regra e
implementação.

## Índice de módulos

| Módulo | Prefixo | Arquivo | Status |
| ------ | ------- | ------- | ------ |
| Plataforma da API (respostas, corpo, CORS, request ID, health) | `PLT` | [modulos/plataforma.md](modulos/plataforma.md) | implementada |
| Usuário e autenticação | `USR` | [modulos/usuario.md](modulos/usuario.md) | implementada |

## Decisões de arquitetura

| ADR | Título | Status |
| --- | ------ | ------ |
| [ADR-0001](adr/0001-autenticacao-por-email-com-jwt.md) | Autenticação por e-mail com JWT | aceita |
| [ADR-0002](adr/0002-padrao-de-respostas-e-erros.md) | Padrão de respostas e erros | aceita |

### Modelo de ADR

```markdown
# ADR-NNNN — Título da decisão

- **Status:** proposta | aceita | substituída por ADR-XXXX
- **Data:** AAAA-MM-DD

## Contexto
Qual problema ou força motivou a decisão.

## Decisão
O que foi decidido, de forma objetiva.

## Consequências
O que fica mais fácil, o que fica mais difícil e o que precisa ser seguido
pelas specs e pelo código.
```
