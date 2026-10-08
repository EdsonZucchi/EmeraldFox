# <Nome do módulo>

| Campo | Valor |
| ----- | ----- |
| Prefixo | `<PREFIXO>` |
| Status | rascunho |
| Última atualização | AAAA-MM-DD |
| Depende de | _specs ou ADRs relacionadas_ |

## 1. Objetivo

Qual problema do usuário este módulo resolve, em poucas linhas.

## 2. Entidades

### <Entidade>

| Campo | Tipo | Obrigatório | Regras |
| ----- | ---- | ----------- | ------ |
| `id` | UUID | sim | Gerado pelo sistema |
| `user_id` | UUID | sim | Dono do registro; nunca informado pelo cliente |
| `created_at` | data/hora UTC | sim | Gerado pelo sistema |

## 3. Regras de negócio

| ID | Regra | Status |
| -- | ----- | ------ |
| `<PREFIXO>-RN-001` | Descrição verificável da regra. | rascunho |

## 4. Operações

### <Ação> — `MÉTODO /rota`

- **Autenticação:** pública | Bearer token
- **Regras aplicadas:** `<PREFIXO>-RN-001`, ...

Entrada:

```json
{}
```

Saída (`200`/`201`):

```json
{ "success": true, "data": {} }
```

Erros:

| Condição | Status | Mensagem |
| -------- | ------ | -------- |
| Exemplo de condição | `422` | Mensagem exata devolvida ao cliente |

## 5. Critérios de aceite

- **`<PREFIXO>-CA-001`** — Dado ..., quando ..., então ...

## 6. Fora de escopo

- O que conscientemente **não** será feito agora.

## 7. Questões em aberto

1. Pergunta que precisa de decisão do usuário (regras afetadas: `<PREFIXO>-RN-00X`).

## 8. Histórico de alterações

| Data | Alteração |
| ---- | --------- |
| AAAA-MM-DD | Criação da spec. |
