# Data Model — Servico-Pagamento

**Fase 1** do `/speckit-plan` · Data: 2026-08-30 · Base: [spec.md](./spec.md) §Key Entities, [research.md](./research.md) D2, D3, D4

## 1. Tabela `transacoes_pagamento`

Única tabela do serviço. O DDL da ERS foi preservado; as três colunas
acrescentadas estão marcadas e justificadas.

```sql
CREATE TABLE transacoes_pagamento (
    id                       VARCHAR(36) PRIMARY KEY,          -- UUID v4 gerado aqui
    reserva_id               VARCHAR(36) NOT NULL UNIQUE,      -- chave de idempotência (D2)
    usuario_id               VARCHAR(36) NOT NULL,             -- `sub` do Keycloak, dono da reserva
    valor_total              DECIMAL(10,2) NOT NULL,
    forma_pagamento          VARCHAR(50) NOT NULL,             -- PIX | CARTAO_CREDITO
    status                   VARCHAR(50) NOT NULL DEFAULT 'PROCESSANDO',
    codigo_transacao_gateway VARCHAR(255),                     -- referência do adquirente
    motivo_falha             TEXT,
    resultado_anunciado      BOOLEAN NOT NULL DEFAULT FALSE,   -- ACRESCENTADA (FR-014, D3)
    pago_em                  TIMESTAMPTZ,                      -- ACRESCENTADA (FR-012)
    criado_em                TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT status_valido CHECK (status IN
        ('PROCESSANDO','PAGO','RECUSADO','CANCELADO','PENDENTE_VERIFICACAO')),
    CONSTRAINT forma_valida CHECK (forma_pagamento IN ('PIX','CARTAO_CREDITO')),
    CONSTRAINT valor_positivo CHECK (valor_total > 0),
    CONSTRAINT pago_em_so_quando_pago CHECK ((status = 'PAGO') = (pago_em IS NOT NULL)),
    CONSTRAINT anuncio_so_apos_estado_final CHECK
        (NOT resultado_anunciado OR status <> 'PROCESSANDO'),
    CONSTRAINT verificacao_nunca_anunciada CHECK
        (NOT (status = 'PENDENTE_VERIFICACAO' AND resultado_anunciado))
);
```

### Colunas acrescentadas ao DDL da ERS

| Coluna | Por quê |
|---|---|
| `resultado_anunciado` | FR-014. É o que fecha a janela entre gravar a transação e publicar o resultado: sem ela, uma queda nesse intervalo faz a reentrega ser ignorada por idempotência e o resultado se perde para sempre. Substitui uma caixa de saída inteira (research D3). |
| `pago_em` | FR-012 exige o instante em que o pagamento foi confirmado, e a ERS o exige no payload de `pagamento.sucesso`. `atualizado_em` não serve: qualquer escrita posterior o move. |
| `status = 'PENDENTE_VERIFICACAO'` | Clarificação Q2 / FR-022. Não é coluna nova, é um valor novo no domínio de `status`. Sem ele, ausência de resposta do adquirente teria de virar `RECUSADO` (nega poltrona possivelmente paga) ou reprocessamento (arrisca cobrança dupla). |

### Índices

Além da chave primária e do índice implícito de `reserva_id UNIQUE`, **nenhum**.
A única consulta do serviço é por `reserva_id` (FR-015), já atendida pela restrição
de unicidade. Índice sobre `usuario_id` ou `status` seria antecipação de requisito
sem consulta que o justifique (princípio I).

### Restrições que carregam invariante

- `reserva_id UNIQUE` é a **garantia de cobrança única** (FR-006). Não é otimização:
  é o mecanismo. `INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *` é o
  que decide, sob concorrência, qual entrega cobra e qual não cobra.
- `pago_em_so_quando_pago` impede transação paga sem instante e instante sem
  pagamento — as duas formas de quebrar o payload de `pagamento.sucesso`.
- `verificacao_nunca_anunciada` torna a FR-010 impossível de violar por engano no
  código: o estado indeterminado não pode ser marcado como anunciado.

## 2. Máquina de estados da transação

```text
                            ┌──────────────┐
                            │ PROCESSANDO  │  (nasce aqui, antes de qualquer cobrança)
                            └──────┬───────┘
             ┌─────────────────────┼─────────────────────┬──────────────────────┐
             ▼                     ▼                     ▼                      ▼
        ┌─────────┐          ┌───────────┐         ┌───────────┐    ┌───────────────────────┐
        │  PAGO   │          │ RECUSADO  │         │ CANCELADO │    │ PENDENTE_VERIFICACAO  │
        └─────────┘          └───────────┘         └───────────┘    └───────────────────────┘
         anunciável           anunciável            anunciável        NUNCA anunciável
        pagamento.sucesso    pagamento.falhou      pagamento.falhou   (quarentena + inspeção)
```

| Estado | Quando | Anúncio | Terminal |
|---|---|---|---|
| `PROCESSANDO` | transação registrada, cobrança ainda não concluída | — | não |
| `PAGO` | adquirente aprovou; `codigo_transacao_gateway` e `pago_em` preenchidos | `pagamento.sucesso` | sim |
| `RECUSADO` | adquirente negou; `motivo_falha` preenchido | `pagamento.falhou` | sim |
| `CANCELADO` | não cobrável — hoje, só reserva expirada (FR-005); nenhuma cobrança foi tentada | `pagamento.falhou` com motivo `RESERVA_EXPIRADA` | sim |
| `PENDENTE_VERIFICACAO` | adquirente não respondeu no prazo; não se sabe se cobrou | **nenhum** | sim |

**Transições permitidas**: apenas `PROCESSANDO → {qualquer um dos outros quatro}`.
Nenhuma transição parte de estado terminal. A guarda vive no domínio e é o que
torna reentrega e ordem invertida inofensivas mesmo que a idempotência falhasse.

`resultado_anunciado` é ortogonal ao estado: vai de `false` a `true` uma única vez,
depois da publicação confirmada, e nunca volta.

## 3. Motivos de falha (`motivo_falha`)

Vocabulário fechado, publicado no campo `motivo` de `pagamento.falhou`:

| Motivo | Estado | Origem |
|---|---|---|
| `RESERVA_EXPIRADA` | `CANCELADO` | prazo da reserva vencido antes da cobrança (FR-005) |
| `SALDO_INSUFICIENTE` | `RECUSADO` | adquirente |
| `CARTAO_RECUSADO` | `RECUSADO` | adquirente |
| `RECUSADO_PELO_ADQUIRENTE` | `RECUSADO` | adquirente, motivo não mapeado |

Anúncio inválido (FR-003) **não** gera transação nem motivo: a mensagem vai para a
fila morta antes de qualquer escrita, porque sem `reserva_id` válido não há chave
de idempotência com que gravar.

## 4. Porta de repositório

Quatro operações, todas sobre a mesma tabela:

| Operação | Semântica |
|---|---|
| `CriarSeAusente(t) → (criada bool, atual Transacao)` | `INSERT ... ON CONFLICT (reserva_id) DO NOTHING RETURNING *`; em conflito, lê a existente. É a porta de entrada da idempotência (D2). |
| `BuscarPorReserva(reservaID) → Transacao` | consulta da FR-015; devolve ausência distinguível de erro. |
| `Finalizar(id, status, codigoGateway, motivo, pagoEm)` | `PROCESSANDO → terminal`, condicionada ao estado atual na cláusula `WHERE`; zero linhas afetadas significa que outro já finalizou. |
| `MarcarAnunciado(id)` | `resultado_anunciado = true`, só a partir de estado terminal anunciável. |

Não há transação de banco de múltiplas instruções no caminho feliz: cada passo é
uma instrução condicionada ao estado esperado, o que dispensa nível de isolamento
especial.

## 5. Entidades sem persistência

- **Intenção de compra** — o `reserva.criada` consumido. Vive só na memória do
  consumo. Deliberadamente **não** é armazenada: `sessao_id` e `poltronas_ids` não
  são lidos por nenhum requisito, e guardá-los seria cópia de estado de que este
  serviço não é dono.
- **Resultado de pagamento** — o fato publicado. Derivado inteiramente da linha da
  transação, e é por isso que a republicação da D3 é possível sem guardar o payload.
