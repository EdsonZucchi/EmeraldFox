# EmeraldFox

REST API em Go para organização financeira pessoal. Stack, estrutura de
camadas, padrão de respostas, erros, logging e o passo a passo para novos
módulos estão no [README.md](README.md).

## Fluxo de trabalho: arquiteto → desenvolvedor

Todo trabalho que envolve regra de negócio segue dois agentes, definidos em
`.claude/agents/`:

| Agente | Papel | Pode escrever em |
| ------ | ----- | ---------------- |
| `arquiteto` | Define e documenta regras de negócio, entidades, endpoints, erros e critérios de aceite. Registra ADRs. | somente `spec/` |
| `desenvolvedor` | Implementa no código as regras `aprovada` da spec, com testes. | tudo, **exceto** `spec/` |

As restrições de escrita são garantidas pelo hook
`.claude/hooks/spec-guard.js`.

Ao receber um pedido de funcionalidade nova ou de mudança de comportamento:

1. Acione o `arquiteto` para criar/atualizar a spec em `spec/modulos/`.
2. Leve ao usuário as *Questões em aberto* que o arquiteto levantar e só
   considere a spec `aprovada` com a confirmação explícita do usuário (o
   arquiteto registra a aprovação).
3. Acione o `desenvolvedor` informando o módulo e os IDs aprovados.
4. Com o relatório do desenvolvedor, acione o `arquiteto` para marcar os IDs
   como `implementada` e tratar as pendências reportadas.

Correções de bug que não mudam regra de negócio, ajustes de infraestrutura e
refatorações podem ir direto ao `desenvolvedor`. Se um bug revelar que a spec
está errada ou incompleta, passe primeiro pelo `arquiteto`.

Convenções da spec (status, IDs, modelo de ADR): [spec/README.md](spec/README.md).

## Comandos

```bash
go run ./cmd/api
go test ./...
go vet ./...
gofmt -l .
```
