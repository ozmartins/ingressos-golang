# Implementation Plan: Ajuda do Terminal por Controller

**Branch**: `005-help-por-controller` (commits direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/005-help-por-controller/spec.md`

## Summary

O `help` passa a mostrar, na seção da API, só os nomes dos controllers (e `health`) mais uma dica, e `<controller> help` mostra as ações daquele controller com sintaxe e descrição. A mudança é pequena e local: o formatador (`formatar.ts`) ganha uma função de ajuda por controller e uma versão resumida da ajuda geral, e o despachante (`despachar.ts`) reconhece `help` como "ação" de cada controller e de `health` antes de resolver a definição. Os dados vêm da tabela declarativa já existente (`tabela.ts`); nada novo é cadastrado. Nenhuma dependência nova, nenhuma requisição à API.

## Technical Context

**Language/Version**: TypeScript estrito, React 19, Node.js 24

**Primary Dependencies**: nenhuma nova (Vitest já presente)

**Storage**: N/A

**Testing**: Vitest: formatadores (`formatarAjuda`, `formatarAjudaController`, `formatarAcoes`) e despachante (`help`, `<controller> help`, `health help`, caixa/espaços, nenhuma chamada ao `fetch`); verificação manual via `quickstart.md`

**Target Platform**: Navegadores modernos (terminal da página `web/`)

**Project Type**: web (frontend single-page, estende `web/`)

**Performance Goals**: Resposta instantânea (lógica local, sem rede)

**Constraints**: Sem alteração nos serviços Go; `help` geral com no máximo 20 linhas (SC-001); demais comandos inalterados (FR-010)

**Scale/Scope**: 2 funções novas/alteradas em `formatar.ts`, 1 helper em `tabela.ts`, 1 ramo novo em `despachar.ts` + testes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Princípio | Conformidade |
|-----------|--------------|
| I. Simplicidade e YAGNI | PASS: reaproveita a tabela de comandos; sem registro de ajuda separado nem biblioteca |
| II. Isolamento do Frontend | PASS: só código de `web/`; nenhuma chamada a serviço |
| III. Tipagem Estrita | PASS: sem `any`; `Controller` já é união literal |
| IV. Verificação Obrigatória | PASS: testes automatizados + `npm run build` + quickstart |
| V. Preservação do Workspace | PASS: edita só arquivos de `src/terminal/` e adiciona `specs/005-*` |

Re-check pós-design: sem violações; Complexity Tracking não necessário.

## Project Structure

### Documentation (this feature)

```text
specs/005-help-por-controller/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/ajuda.md   # formato exato das saídas de ajuda
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
src/terminal/
├── despachar.ts             # EDIT: `<controller> help` e `health help` (sem requisição)
├── despachar.test.ts        # EDIT: casos novos; atualizar as expectativas de `filmes`/`filmes xpto`
└── api/
    ├── tabela.ts            # EDIT: helper com os nomes digitáveis (controllers CRUD + health)
    ├── formatar.ts          # EDIT: formatarAjuda resumido; formatarAjudaController; dica em formatarAcoes
    └── formatar.test.ts     # EDIT: testes dos três formatadores
```

**Structure Decision**: Mantém o layout das features 003/004. A ajuda continua sendo responsabilidade do formatador; o despachante só decide qual formatador chamar.

## Complexity Tracking

Sem violações a justificar.
