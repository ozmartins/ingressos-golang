# Implementation Plan: Comandos de Escrita do Terminal Guiados por Campos

**Branch**: `004-campos-comandos-escrita` (commits direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-campos-comandos-escrita/spec.md`

## Summary

Os oito comandos de escrita que hoje recebem JSON (`criar` e `atualizar` de filmes, cinemas, salas e sessões) passam a abrir um **diálogo de preenchimento** no terminal: uma pergunta por campo, validação local, resumo e confirmação, e só então a aplicação monta o corpo e o envia pelo mesmo caminho de requisição já existente (feature 003). O diálogo é uma **máquina de estados pura** (sem React, testável com Vitest) alimentada por **formulários declarativos** (um por combinação controller + ação, espelhando os schemas `*Entrada`/`*Atualizacao` do OpenAPI do catálogo). O componente `Terminal` apenas guarda o estado do diálogo e, enquanto ele existe, encaminha as linhas digitadas ao diálogo em vez do despachante de comandos, sem gravá-las no histórico. Nenhuma dependência nova; nenhuma alteração nos serviços Go.

## Technical Context

**Language/Version**: TypeScript estrito, React 19, Node.js 24

**Primary Dependencies**: nenhuma nova (Vitest já presente)

**Storage**: N/A — estado do diálogo apenas em memória (estado do componente)

**Testing**: Vitest: conversão/validação de cada tipo de campo, formulários dos 8 comandos (campos, obrigatoriedade, corpo montado), máquina de estados (avançar, repetir pergunta inválida, pular opcional, fileiras, resumo, confirmar, cancelar), despachante (autenticação antes do diálogo, recusa de JSON na linha, envio após confirmação com `fetch` falso); verificação manual via `quickstart.md`

**Target Platform**: Navegadores modernos; catálogo em `localhost:8082` via proxy do Vite (inalterado)

**Project Type**: web (frontend single-page, estende `web/`)

**Performance Goals**: Cada resposta validada e a próxima pergunta exibida de forma instantânea (lógica local, sem rede até a confirmação)

**Constraints**: Sem alterações nos serviços Go; contrato da API respeitado (`additionalProperties: false`, tipos, enums, limites); respostas do diálogo fora do histórico de comandos; comandos de leitura, `remover`, `health`, `login`, `logout` inalterados

**Scale/Scope**: 8 formulários (4 de criação, 4 de atualização), ~3 arquivos-fonte novos (`api/formularios.ts`, `dialogo.ts` e testes) e edições em `tabela.ts`, `argumentos.ts`, `despachar.ts`, `formatar.ts`, `Terminal.tsx`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Princípio | Conformidade |
|-----------|--------------|
| I. Simplicidade e YAGNI | PASS: formulários declarativos + uma máquina de estados pura; sem biblioteca de formulários/CLI; sem pré-preenchimento na atualização (fora do escopo) |
| II. Isolamento do Frontend | PASS: só o contrato REST do catálogo; nenhum código Go alterado |
| III. Tipagem Estrita | PASS: tipos de campo como união discriminada; valores montados como `unknown` estreitado; sem `any` |
| IV. Verificação Obrigatória | PASS: testes automatizados da lógica + `npm run build` + quickstart manual |
| V. Preservação do Workspace | PASS: só edita arquivos de `src/terminal/` e adiciona `specs/004-*` |

Re-check pós-design: sem violações; Complexity Tracking não necessário.

## Project Structure

### Documentation (this feature)

```text
specs/004-campos-comandos-escrita/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── formularios.md   # campos pedidos por comando ↔ schemas do OpenAPI
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
src/terminal/
├── Terminal.tsx            # EDIT: estado do diálogo; roteia a linha digitada; não grava respostas no histórico; Esc cancela
├── despachar.ts            # EDIT: auth antes do diálogo; inicia diálogo; continuarDialogo(); envio extraído e reutilizado
├── dialogo.ts              # NEW: máquina de estados pura (iniciar, responder, cancelar)
├── dialogo.test.ts         # NEW
├── despachar.test.ts       # EDIT: casos de escrita pelo novo fluxo
└── api/
    ├── formularios.ts      # NEW: definições de campo, FORMULARIOS, converter(), montarCorpo()
    ├── formularios.test.ts # NEW
    ├── tabela.ts           # EDIT: `uso` sem <json>
    ├── argumentos.ts       # EDIT: remove leitura de JSON; criar sem args, atualizar só com id; JSON na linha → erro orientando
    ├── argumentos.test.ts  # EDIT
    └── formatar.ts         # EDIT: ajuda/uso sem exemplos JSON
```

**Structure Decision**: Mantém o layout da feature 003 (`src/terminal/` + `src/terminal/api/`); a lógica de diálogo fica em `dialogo.ts` ao lado de `despachar.ts` por ser lógica de terminal, e a descrição dos campos em `api/formularios.ts` por espelhar o contrato da API.

## Complexity Tracking

Sem violações a justificar.
