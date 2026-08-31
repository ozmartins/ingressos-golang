# Data Model: Emissão e Validação de Ingressos Digitais

**Fase 1** | **Data**: 2026-08-30 | **Spec**: [spec.md](./spec.md) | **Decisões**: [research.md](./research.md)

Duas tabelas, exatamente as da ERS (research D10). O que este documento acrescenta ao
DDL da ERS são as restrições `CHECK` que trazem invariantes do domínio para dentro do
banco, e um índice para a listagem.

---

## 1. `ingressos_emitidos`

```sql
CREATE TABLE ingressos_emitidos (
    id           VARCHAR(36)  PRIMARY KEY,
    reserva_id   VARCHAR(36)  NOT NULL UNIQUE,
    usuario_id   VARCHAR(36)  NOT NULL,
    codigo_qr    VARCHAR(255) NOT NULL UNIQUE,
    status       VARCHAR(50)  NOT NULL DEFAULT 'VALIDO',
    utilizado_em TIMESTAMPTZ,
    criado_em    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT status_valido CHECK (status IN ('VALIDO','UTILIZADO','CANCELADO')),
    CONSTRAINT utilizado_em_so_quando_utilizado CHECK
        ((status = 'UTILIZADO') = (utilizado_em IS NOT NULL))
);

CREATE INDEX ingressos_por_pessoa ON ingressos_emitidos (usuario_id, criado_em DESC);
```

| Coluna | Origem | Papel |
|---|---|---|
| `id` | gerado (UUID v4) | identidade opaca; é o que viaja assinado dentro do `codigo_qr` (D3) |
| `reserva_id` | `pagamento.sucesso` | **a chave da idempotência**. O `UNIQUE` é o mecanismo da FR-004, não um adorno (D2) |
| `usuario_id` | `pagamento.sucesso` | dono; recorta a listagem (FR-014) |
| `codigo_qr` | derivado de `id` | o que a portaria lê. `UNIQUE` decorre de `id` ser único, mas é declarado para que o banco também o garanta |
| `status` | domínio | `VALIDO` na emissão |
| `utilizado_em` | domínio | instante da baixa; imutável depois de gravado (FR-008) |
| `criado_em` | relógio injetado | chave de ordenação da listagem (FR-023) |

**Por que estas restrições `CHECK`**

- `status_valido` — o vocabulário de estados é fechado (FR-019). Sem isso, um `UPDATE`
  errado grava um estado que o domínio não conhece e a validação passa a responder de
  forma imprevisível.
- `utilizado_em_so_quando_utilizado` — é a invariante da FR-007 e da FR-008 escrita
  como igualdade nos dois sentidos: ingresso utilizado **tem** instante, e ingresso
  não utilizado **não** tem. Impede tanto a baixa sem carimbo quanto o carimbo sem baixa.

**O que deliberadamente não existe aqui**

- Nenhuma coluna de valor, sessão, filme ou poltrona: nada nesta feature os usa, e o
  `valor_total` que chega no anúncio é descartado (D1).
- Nenhuma coluna equivalente ao `resultado_anunciado` do `Servico-Pagamento`: este
  serviço não publica fato nenhum, então não há janela entre gravar e publicar a fechar
  (D10).
- Nenhum índice por `status`: o recorte por pessoa já reduz o conjunto a dezenas de
  linhas (D8).

---

## 2. Máquina de estados do ingresso

```
                   emissão
                      │
                      ▼
                  ┌────────┐
                  │ VALIDO │
                  └───┬────┘
             baixa    │    cancelamento
          (FR-007)    │    (sem gatilho nesta feature)
              ┌───────┴───────┐
              ▼               ▼
       ┌────────────┐   ┌───────────┐
       │ UTILIZADO  │   │ CANCELADO │
       └────────────┘   └───────────┘
          terminal         terminal
```

| De | Para | Gatilho | Nesta feature |
|---|---|---|---|
| — | `VALIDO` | anúncio de pagamento confirmado | **sim** |
| `VALIDO` | `UTILIZADO` | validação na portaria | **sim** |
| `VALIDO` | `CANCELADO` | — | **não há gatilho** (clarificação 2) |
| terminal | qualquer | — | rejeitado pelo domínio (FR-019) |

`CANCELADO` está no modelo porque a validação precisa saber negá-lo e a listagem precisa
saber exibi-lo, mas **nenhuma operação desta feature leva um ingresso até ele**. O
domínio ainda assim rejeita qualquer saída de `CANCELADO`, e essa rejeição é testada —
é a defesa que já estará no lugar quando o gatilho de cancelamento existir.

**Consequência para a restrição `utilizado_em_so_quando_utilizado`**: como `UTILIZADO`
é terminal, um ingresso utilizado nunca vira cancelado, e a restrição nunca é violada
por transição legítima.

---

## 3. `registros_notificacao`

```sql
CREATE TABLE registros_notificacao (
    id          VARCHAR(36) PRIMARY KEY,
    ingresso_id VARCHAR(36) NOT NULL REFERENCES ingressos_emitidos(id),
    usuario_id  VARCHAR(36) NOT NULL,
    canal       VARCHAR(50) NOT NULL DEFAULT 'EMAIL',
    status      VARCHAR(50) NOT NULL DEFAULT 'ENVIADO',
    detalhes    TEXT,
    enviado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT canal_valido  CHECK (canal  IN ('EMAIL','PUSH','SMS')),
    CONSTRAINT status_valido CHECK (status IN ('ENVIADO','FALHA')),
    CONSTRAINT detalhes_na_falha CHECK (status <> 'FALHA' OR detalhes IS NOT NULL)
);
```

`detalhes_na_falha` é a única restrição além do vocabulário: a FR-017 exige "o detalhe
em caso de falha", e um registro de falha sem motivo não serve ao propósito de reenvio
que justifica a tabela existir. Em `ENVIADO`, `detalhes` fica nulo.

Vários registros por ingresso são possíveis (a chave estrangeira não é única) — é o que
permitirá a uma feature futura de reenvio acrescentar tentativas sem apagar o histórico.
Nesta feature, cada ingresso produz **exatamente um** registro, porque o aviso é
disparado uma única vez, no momento da emissão (D6).

**Nenhum índice** além da chave primária: nada nesta feature consulta esta tabela. Ela
é escrita no processamento e lida por gente, em investigação. Índice para consulta que
não existe é complexidade sem carga.

---

## 4. Composição do `codigo_qr` (D3)

```
CIN1.<base64url(id)>.<base64url(HMAC-SHA256(segredo, "CIN1." + base64url(id)))>
```

- **Geração** (emissão): sorteia o `id`, monta as duas primeiras partes, assina, grava
  o resultado inteiro na coluna.
- **Verificação** (validação): confere o prefixo e a forma de três partes; decodifica o
  `id`; recalcula o HMAC sobre as duas primeiras partes; compara em tempo constante.
  Falhou qualquer etapa → recusa **sem tocar no banco** (FR-010).
- **Consulta**: só um código com assinatura válida vira `WHERE id = $1`.

A resposta é a mesma para código malformado, assinatura inválida e ingresso inexistente
— indistinguíveis por exigência da FR-010.

---

## 5. Porta de repositório

Assinaturas conceituais; os tipos concretos são do implementador.

```go
// Ingressos é a porta de persistência do ingresso.
type Ingressos interface {
    // CriarSeAusente é a porta de entrada da idempotência (FR-004, D2).
    // criado=false significa que outra entrega chegou primeiro: esta NÃO emite
    // e NÃO avisa.
    CriarSeAusente(ctx, ing ingresso.Ingresso) (criado bool, atual ingresso.Ingresso, err error)

    // Utilizar é a escrita condicionada da D4: aplica a baixa somente se o
    // ingresso estiver VALIDO. autorizado=false NÃO diz por quê.
    Utilizar(ctx, id string, agora time.Time) (autorizado bool, err error)

    // BuscarPorID existe para explicar uma recusa, depois de Utilizar devolver
    // false. Fora do caminho de sucesso.
    BuscarPorID(ctx, id string) (ingresso.Ingresso, error)

    // ListarPorUsuario aplica ordenação e filtro da FR-023 e FR-024. Filtro
    // vazio significa todos os estados.
    ListarPorUsuario(ctx, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error)
}

// Avisos é a porta de persistência do registro de notificação.
type Avisos interface {
    Registrar(ctx, r aviso.Registro) error
}

// Notificador é a porta de saída do aviso (D6). O único adaptador desta entrega
// é o simulado; a falha dele é capturada, nunca propagada (FR-025).
type Notificador interface {
    Avisar(ctx, ing ingresso.Ingresso) error
}
```

`Utilizar` devolver apenas um booleano é deliberado: o caso de uso não deve poder
confundir "autorizei" com "encontrei". O motivo da recusa é uma segunda pergunta, feita
só quando a primeira já respondeu não.

---

## 6. Rastreabilidade requisito → modelo

| Requisito | Onde vive |
|---|---|
| FR-003 | colunas de `ingressos_emitidos` |
| FR-004 | `UNIQUE (reserva_id)` + `CriarSeAusente` |
| FR-005, FR-006 | §4, composição do `codigo_qr` |
| FR-007, FR-008 | `Utilizar` + `utilizado_em_so_quando_utilizado` |
| FR-009, FR-010 | §4 (recusa antes do banco) + `status_valido` |
| FR-011 | `Utilizar` como escrita condicionada (D4) |
| FR-013, FR-014 | `ListarPorUsuario` recortado por `usuario_id` |
| FR-017, FR-018 | `registros_notificacao` + `detalhes_na_falha` |
| FR-019, FR-020 | §2 + colunas nunca reescritas após a emissão |
| FR-023, FR-024 | `ORDER BY criado_em DESC, id DESC` + índice `ingressos_por_pessoa` |
