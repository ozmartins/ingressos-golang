# Implementation Plan: Aplicação React Vazia

**Branch**: `001-empty-react-app` | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-empty-react-app/spec.md`

## Summary

Criar, em `ingressos-golang/web/`, uma aplicação React mínima (Vite + React + TypeScript, gerenciada com npm) que renderiza apenas um contêiner raiz vazio, inicia em modo de desenvolvimento e gera build de produção sem erros. Sem roteamento, UI, estado, autenticação, chamadas a serviços nem Docker. Arquivos existentes (`.claude/`, `.specify/`, `.gitignore`, `.dockerignore`, `specs/`) são preservados.

## Technical Context

**Language/Version**: TypeScript (modo estrito) sobre Node.js 24 (instalado: v24.22.3)

**Primary Dependencies**: react 19, react-dom 19; dev: vite 8, @vitejs/plugin-react, typescript, @types/react, @types/react-dom

**Storage**: N/A

**Testing**: Sem framework de testes nesta etapa (app vazia, YAGNI). Verificação por `tsc` + `vite build` e inspeção manual descrita em `quickstart.md`

**Target Platform**: Navegadores modernos; desenvolvimento em Linux/Node.js

**Project Type**: web (frontend single-page)

**Performance Goals**: Servidor de desenvolvimento pronto em poucos segundos; build de produção em menos de 1 minuto

**Constraints**: Zero erros/avisos no console; preservar arquivos existentes; sem dependências além do necessário

**Scale/Scope**: 1 página vazia, ~4 arquivos-fonte

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constituição v1.0.0 (`.specify/memory/constitution.md`):

| Princípio | Conformidade |
|-----------|--------------|
| I. Simplicidade e YAGNI | PASS: apenas react, react-dom e o ferramental mínimo; sem UI, rotas ou estado |
| II. Isolamento do Frontend | PASS: `web/` independente, sem integração com os serviços Go |
| III. Tipagem Estrita | PASS: TypeScript com `strict` (T003) |
| IV. Verificação Obrigatória | PASS: `npm run build` (tsc + vite) e console limpo (T010–T012); sem lógica de negócio, logo sem testes automatizados |
| V. Preservação do Workspace | PASS: FR-005 e T013 |

Pós-design: sem mudanças. **Resultado: PASS.**

## Project Structure

### Documentation (this feature)

```text
specs/001-empty-react-app/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

`contracts/` não é gerado: a aplicação não expõe interfaces externas.

### Source Code (repository root = `ingressos-golang/web/`)

```text
web/
├── index.html
├── package.json
├── package-lock.json
├── tsconfig.json
├── tsconfig.node.json
├── README.md
├── vite.config.ts
├── .gitignore          # já existente (node_modules, dist)
└── src/
    ├── main.tsx        # monta <App /> em #root
    ├── App.tsx         # componente raiz vazio
    └── vite-env.d.ts
```

**Structure Decision**: Projeto único na raiz de `web/`, seguindo o layout padrão do Vite (o mesmo usado pelo frontend removido no commit `3d59c37`). Sem `Dockerfile`/`nginx.conf`.

## Complexity Tracking

Sem violações.
