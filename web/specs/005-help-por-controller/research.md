# Research: Ajuda por Controller

## D1 — Onde reconhecer `<controller> help`
- **Decisão**: no ramo de controllers do `despachar`, antes de `buscarDefinicao`: se a ação (em minúsculas) for `help`, devolver `formatarAjudaController(controller)`. Para `health`, tratar `help` em `resto` antes de `executarDefinicao`.
- **Racional**: `help` não é uma ação da tabela (não tem método/caminho); tratá-lo antes evita qualquer requisição (FR-006) e não polui `TABELA`, que tem testes de unicidade das 21 ações.
- **Alternativas**: adicionar `help` à tabela (quebra a contagem de 21 e as regras de `usaId`/`protegido`); descartado.

## D2 — Conteúdo da ajuda do controller
- **Decisão**: reutilizar `definicoesDe(controller)` e o mesmo par de linhas do help atual (`uso` em `saida`, `descricao` recuada em `sistema`), mais, nos controllers com escrita, a nota de que `criar`/`atualizar` perguntam os campos e podem ser abortados com `cancelar`/Esc (nota que hoje está no help geral).
- **Racional**: FR-004 pede o mesmo formato de hoje; a nota só faz sentido junto das ações de escrita e libera linhas no help geral (SC-001).

## D3 — Lista resumida
- **Decisão**: nomes vindos de `CONTROLLERS_CRUD` + `health`, um por linha, recuados em 2 espaços, seguidos da dica `Digite "<controller> help" para ver as opções (ex.: filmes help).`.
- **Racional**: FR-001/FR-002. `health` entra pelo nome por ser o único comando do grupo operacional (Assumptions).

## D4 — `help <texto>`
- **Decisão**: sem mudança: o ramo `chave === 'help'` já ignora o restante e mostra o help geral (FR-009).

## D5 — Mensagens de ação ausente/desconhecida
- **Decisão**: `formatarAcoes` ganha uma segunda linha `use: <controller> help para ver os detalhes.` (FR-008); as expectativas existentes de `despachar.test.ts` e `formatar.test.ts` são atualizadas.

## D6 — Meta de 20 linhas
- **Decisão**: contar as linhas do help geral em teste (`<= 20`).
- **Racional**: SC-001 vira verificação automatizada.
