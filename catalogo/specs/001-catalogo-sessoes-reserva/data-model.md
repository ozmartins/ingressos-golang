# Phase 1 — Data Model: Catálogo de Filmes, Sessões e Reserva de Poltronas

**Feature**: `001-catalogo-sessoes-reserva` | **Date**: 2026-08-29

O esquema segue a DDL da ERS sem alterações. Este documento acrescenta o que a DDL não expressa: invariantes, transições de estado, o mapeamento para os tipos do domínio e as consultas que sustentam SC-003.

## Visão geral

```text
Filme 1 ────< Sessao >──── 1 Sala >──── 1 Cinema
                 │
                 └── (referência lógica, sem FK) ──> Servico-Estoque
                                                     poltronas, bloqueios, expiração
```

Este serviço é **somente leitura** sobre as quatro tabelas. Nenhuma entidade de reserva é persistida aqui (FR-031).

---

## Filme

Obra em exibição ou prevista.

| Campo | Tipo (banco) | Tipo (domínio) | Regras |
|---|---|---|---|
| `id` | `VARCHAR(36)` PK | `string` (UUID v4) | Identidade estável, exposta ao cliente |
| `titulo` | `VARCHAR(255)` NOT NULL | `string` | Obrigatório |
| `sinopse` | `TEXT` | `*string` | Opcional — ausência não pode quebrar a listagem |
| `duracao_minutos` | `INT` NOT NULL | `int` | > 0 |
| `classificacao_etaria` | `VARCHAR(50)` NOT NULL | `string` | Texto livre (ex.: "14 anos") |
| `genero` | `VARCHAR(100)` NOT NULL | `string` | Obrigatório |
| `imagem_url` | `VARCHAR(500)` | `*string` | Opcional |
| `status` | `VARCHAR(50)` NOT NULL | `StatusFilme` | `EM_CARTAZ` \| `BREVE` \| `FORA_DE_CARTAZ` |
| `criado_em` / `atualizado_em` | `TIMESTAMPTZ` | — | Não expostos ao cliente |

**Invariantes**
- `status` fora do conjunto conhecido é erro de dados, não valor a repassar: a leitura falha alto em vez de vazar valor desconhecido ao cliente.
- Campos opcionais nulos são omitidos da resposta, nunca substituídos por string vazia (edge case "material de apoio ausente").

**Regra de exposição (FR-008)**: sem filtro, a listagem pública retorna apenas `EM_CARTAZ` e `BREVE`. `FORA_DE_CARTAZ` só aparece se explicitamente pedido — e a spec não exige oferecê-lo no filtro público; o valor é aceito, mas listado como tal.

---

## Cinema

Complexo físico onde ocorrem as exibições.

| Campo | Tipo (banco) | Tipo (domínio) | Regras |
|---|---|---|---|
| `id` | `VARCHAR(36)` PK | `string` (UUID v4) | |
| `nome` | `VARCHAR(255)` NOT NULL | `string` | |
| `cidade` | `VARCHAR(100)` NOT NULL | `string` | |
| `estado` | `VARCHAR(2)` NOT NULL | `string` | Sigla de 2 letras |
| `endereco` | `TEXT` NOT NULL | `string` | |
| `criado_em` | `TIMESTAMPTZ` | — | Não exposto |

**Relacionamento**: agrupa 0..N salas.

---

## Sala

Ambiente de exibição dentro de um cinema.

| Campo | Tipo (banco) | Tipo (domínio) | Regras |
|---|---|---|---|
| `id` | `VARCHAR(36)` PK | `string` (UUID v4) | |
| `cinema_id` | `VARCHAR(36)` FK NOT NULL | `string` | Pertence a exatamente um cinema |
| `numero` | `INT` NOT NULL | `int` | Único dentro do cinema (invariante de negócio; a DDL não a impõe) |
| `tipo_tela` | `VARCHAR(50)` NOT NULL | `TipoTela` | `2D` \| `3D` \| `IMAX` \| `VIP` |
| `capacidade_total` | `INT` NOT NULL | `int` | > 0 |
| `criado_em` | `TIMESTAMPTZ` | — | Não exposto |

**Nota**: `capacidade_total` é informativa para o cliente. **Não** é usada para validar poltronas — o mapa de assentos pertence ao estoque (premissa da spec).

---

## Sessao

Exibição de um filme em uma sala num instante específico. É o agregado central da feature.

| Campo | Tipo (banco) | Tipo (domínio) | Regras |
|---|---|---|---|
| `id` | `VARCHAR(36)` PK | `string` (UUID v4) | |
| `filme_id` | `VARCHAR(36)` FK NOT NULL | `string` | |
| `sala_id` | `VARCHAR(36)` FK NOT NULL | `string` | |
| `data_hora_inicio` | `TIMESTAMPTZ` NOT NULL | `time.Time` (UTC) | Sempre em UTC no domínio |
| `idioma` | `VARCHAR(50)` NOT NULL | `Idioma` | `DUBLADO` \| `LEGENDADO` |
| `preco_base` | `DECIMAL(10,2)` NOT NULL | decimal exato | **Nunca** `float64` — ver D2 em research.md |
| `status` | `VARCHAR(50)` NOT NULL | `StatusSessao` | `AGENDADA` \| `EM_ANDAMENTO` \| `FINALIZADA` \| `CANCELADA` |
| `criado_em` | `TIMESTAMPTZ` | — | Não exposto |

### Transições de estado

```text
AGENDADA ──> EM_ANDAMENTO ──> FINALIZADA
    │
    └──> CANCELADA
```

As transições são executadas pelo processo administrativo externo; este serviço apenas as observa. O que ele decide a partir delas:

| Situação | Aparece na grade pública (FR-016) | Aceita reserva (FR-022) |
|---|---|---|
| `AGENDADA` | Sim | Sim, se `data_hora_inicio` ainda for futura |
| `EM_ANDAMENTO` | Sim | Não — a sessão já começou |
| `FINALIZADA` | Não | Não |
| `CANCELADA` | Não | Não |

**Invariante de reserva**: uma sessão só aceita reserva se `status = AGENDADA` **e** `data_hora_inicio > agora`. A segunda condição existe porque a transição para `EM_ANDAMENTO` é feita por um processo externo e pode atrasar — confiar apenas no status deixaria uma janela em que sessões já iniciadas aceitariam reservas (edge case "o relógio cruza o momento presente").

### Vista de exibição (`SessaoDetalhada`)

A grade expõe dados de quatro tabelas em uma só linha. O domínio modela isso como um objeto de leitura, não como agregado com identidade própria:

`id`, `filme_id`, `filme_titulo`, `cinema_id`, `cinema_nome`, `sala_numero`, `tipo_tela`, `data_hora_inicio`, `idioma`, `preco_base`.

**Edge case coberto**: se a junção não encontrar filme ou sala (dado removido ou inconsistente), a sessão é **omitida** da grade e o fato é registrado como aviso — a consulta não retorna registro incompleto nem falha inteira (edge case correspondente na spec). A junção interna (`INNER JOIN`) faz isso naturalmente; o aviso vem de uma verificação de contagem.

---

## Paginação (`Page` / `PageRequest`)

Tipos compartilhados em `internal/domain/shared`, usados por todas as consultas de coleção.

```text
PageRequest{ Numero int (>=1), Tamanho int (1..max) }
Page[T]{ Itens []T, Total int, Pagina int, Tamanho int, TemProxima bool }
```

**Regras**
- `Numero` padrão 1; `Tamanho` padrão 20, teto 100 (configurável).
- `Tamanho` acima do teto é **recusado** com erro de validação, não silenciosamente reduzido — SC-008 exige que o teto seja observável, e reduzir em silêncio esconde do cliente que ele pediu algo inválido.
- `TemProxima` = `Numero * Tamanho < Total`.
- Posição além do fim: `Itens` vazio, `Total` correto, sem erro (FR-005).

**Ordenação determinística por coleção** (FR-004):

| Coleção | `ORDER BY` |
|---|---|
| Filmes | `titulo ASC, id ASC` |
| Cinemas | `nome ASC, id ASC` |
| Salas | `numero ASC, id ASC` |
| Sessões | `data_hora_inicio ASC, id ASC` |

O desempate por `id` é o que torna a ordem total, e não apenas parcial — sem ele, dois filmes de mesmo título poderiam trocar de posição entre duas páginas.

---

## SolicitacaoReserva / ResultadoReserva

Objetos de domínio **transientes** — nunca persistidos (FR-031).

```text
SolicitacaoReserva{ SessaoID string, PoltronasIDs []string, UsuarioID string }
```

**Validações antes de qualquer chamada ao estoque** (FR-023, FR-021, FR-022):
1. `UsuarioID` não vazio — vem da claim `sub`; ausência é recusa, não anônimo.
2. `PoltronasIDs` não vazia.
3. `PoltronasIDs` sem duplicatas (comparação exata, sensível a caixa).
4. `SessaoID` existe e a sessão aceita reserva (ver invariante acima).

Falhar qualquer uma dessas evita uma ida à rede — é o que faz os cenários 3, 5 e 6 da User Story 3 não contatarem o estoque.

```text
ResultadoReserva{ Sucesso bool, ReservaID string, Mensagem string, ExpiraEm time.Time }
```

**Invariante de integridade**: `Sucesso == true` exige `ReservaID` não vazio **e** `ExpiraEm` válido. Uma resposta do estoque que afirme sucesso sem esses dados é tratada como falha de contrato do parceiro (502), nunca repassada como sucesso ao cliente (edge case explícito na spec).

---

## Consultas e índices

As quatro consultas de listagem seguem a mesma forma:

```sql
-- 1) a página
SELECT <colunas>
FROM <tabela> [JOINs]
WHERE <filtros opcionais>
ORDER BY <chave>, id
LIMIT $n OFFSET $m;

-- 2) o total
SELECT COUNT(*) FROM <tabela> [JOINs] WHERE <filtros opcionais>;
```

São duas consultas, e isso é deliberado. Trazer o total na mesma via
`COUNT(*) OVER ()` obriga o planejador a materializar todo o conjunto filtrado
antes de ordenar, o que descarta o índice de ordenação — medido com
`EXPLAIN ANALYZE` durante a implementação. A primeira consulta passa a ser
proporcional ao tamanho da página; a segunda é o custo linear inevitável do
total exato.

**Índices necessários** (além das PKs e FKs da DDL), para sustentar SC-003:

| Índice | Serve a |
|---|---|
| `filmes (titulo, id)` | **Ordenação** da listagem de filmes |
| `cinemas (nome, id)` | **Ordenação** da listagem de cinemas |
| `salas (cinema_id, numero, id)` | Filtro e ordenação das salas de um cinema |
| `sessoes (data_hora_inicio, id)` | **Ordenação** da grade |
| `filmes (status)` | Filtro seletivo por situação e contagem |
| `sessoes (status, data_hora_inicio)` | Filtro por situação |
| `sessoes (filme_id, data_hora_inicio)` | Grade filtrada por filme |
| `sessoes (sala_id)` | Junção com sala e filtro por cinema |

Os índices de ordenação começam pela chave do `ORDER BY`, não pelo filtro. A
versão inicial deste plano fazia o contrário, e a medição mostrou por que não
funciona: o filtro por situação seleciona quase toda a tabela, então um índice
que comece por `status` não ajuda a ordenar e o planejador o descarta. Começando
pela chave de ordenação, o banco percorre o índice em ordem, aplica o filtro
linha a linha e para nas primeiras que passarem.

O filtro por cinema alcança `sessoes` através de `salas`, então o índice em `salas (cinema_id, ...)` é o que evita varredura nessa combinação. Esses índices entram como migração própria — a DDL da ERS não os inclui.

**Filtro de data**: `data` (YYYY-MM-DD) é interpretada como o intervalo `[data 00:00, data+1 00:00)` no fuso de exibição, comparado contra `data_hora_inicio`. Usar intervalo em vez de `DATE(data_hora_inicio) = $1` preserva o uso do índice — uma função sobre a coluna o descartaria.
