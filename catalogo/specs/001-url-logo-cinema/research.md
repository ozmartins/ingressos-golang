# Research: URL do logo do cinema

## D1 — Semântica do PUT sem `logo_url`
- **Decision**: substituição completa; omitir ou enviar `null`/`""` remove o logo.
- **Rationale**: `PutCinema` já usa `NovoCinema` com todos os campos e `Atualizar` grava todas as colunas; mesma regra de `ativo` (default). Manter PUT parcial exigiria distinguir "ausente" de "vazio" e quebraria a consistência do recurso.
- **Alternatives**: preservar quando ausente (exigiria merge com leitura prévia ou ponteiro duplo); PATCH novo (fora de escopo).

## D2 — Tipo e tamanho da coluna
- **Decision**: `VARCHAR(2048) NULL`, sem default; linhas antigas ficam `NULL`.
- **Rationale**: limite espelhado no banco, adição sem reescrita da tabela (nullable sem default é metadado-apenas no PostgreSQL).
- **Alternatives**: `NOT NULL DEFAULT ''` (reescreve a tabela em versões antigas, mistura vazio e ausente).

## D3 — Representação no domínio
- **Decision**: `LogoURL string` em `Cinema` e `DadosCinema`; `""` = sem logo. Repositório grava `NULL` quando vazio e lê `NULL` como `""`.
- **Rationale**: o domínio não depende de tipos de banco; mantém o padrão dos demais campos string.

## D4 — Validação
- **Decision**: em `NovoCinema`: `TrimSpace`; vazio → sem logo; senão `net/url.Parse`, esquema `http`/`https` (case-insensitive), `Host` não vazio, sem espaços internos, `len ≤ 2048`; falhas viram `shared.ErrValidacao` com mensagem `logo_url ...`.
- **Rationale**: biblioteca padrão, sem dependência nova; bloqueia `javascript:`/`data:`.
- **Alternatives**: aceitar qualquer string (risco de XSS ao ser renderizado no front).

## D5 — Contrato de saída
- **Decision**: `logo_url` sempre presente na resposta, `null` quando não há logo; campo opcional na entrada (`additionalProperties: false` mantido, então precisa constar em `CinemaEntrada`).
- **Rationale**: frontend distingue ausência sem checar existência da chave; não quebra clientes (campo novo, não obrigatório).

## D6 — Migração
- **Decision**: `000007_logo_do_cinema` com `ALTER TABLE catalogo.cinemas ADD COLUMN logo_url VARCHAR(2048)`; down com `DROP COLUMN IF EXISTS`.
- **Rationale**: segue 000003; reversível (FR-006).
