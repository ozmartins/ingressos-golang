# Data Model: Ajuda por Controller

Sem novas entidades persistidas. Reaproveita a tabela existente (`Definicao` em `src/terminal/api/tabela.ts`).

| Conceito | Origem | Uso |
|----------|--------|-----|
| Controller | `Controller` / `CONTROLLERS_CRUD` | nomes na lista resumida e argumento de `formatarAjudaController` |
| Opção de controller | `Definicao` (`uso`, `descricao`) de `definicoesDe(controller)` | linhas de `<controller> help` |
| Comando `health` | `Definicao` com `acao: 'health'` | linha na lista resumida e conteúdo de `health help` |

Novo helper: `nomesDeComandos(): string[]` → `['filmes', 'cinemas', 'salas', 'sessoes', 'health']`.
