# Implementation Plan: Página inicial com abas de Filmes e Cinemas

**Branch**: `001-home-filmes-cinemas` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-home-filmes-cinemas/spec.md`

## Summary

Substituir o `App` vazio do `front/` por uma página inicial com duas abas (Filmes, Cinemas). Os dados vêm das listagens públicas do serviço `catalogo` (`GET /api/v1/filmes` e `GET /api/v1/cinemas`), carregados por completo (percorrendo as páginas do servidor) e filtrados no navegador por funções puras. A aba ativa vive na query string (`?aba=`), os filtros de cada aba vivem em estado da página e sobrevivem à troca de aba. O catálogo não envia CORS, então o acesso passa pelo proxy do Vite, como já faz o `admin/`.

## Technical Context

**Language/Version**: TypeScript ~7.0 (já no `package.json`), React 19.3

**Primary Dependencies**: React, Vite 8 (existentes); nenhuma biblioteca nova de runtime (sem router, sem gerenciador de dados)

**Storage**: N/A (somente leitura da API; filtros em memória)

**Testing**: Vitest (mesma escolha do `admin/`) para lógica pura (filtros, normalização, paginação do cliente); testes de componente com Testing Library e jsdom para abas e estados; `oxlint` para lint

**Target Platform**: Navegadores modernos, desktop e celular (≥ 360 px)

**Project Type**: Web application (SPA frontend, consome serviço existente)

**Performance Goals**: Lista visível em ≤ 3 s; atualização de filtro em < 1 s (SC-001, SC-003)

**Constraints**: Catálogo sem CORS (proxy do Vite em `/catalogo`); tamanho máximo de página do catálogo é 100 (`PAGINACAO_TAMANHO_MAXIMO`); sem login

**Scale/Scope**: Até 500 filmes e 200 cinemas (SC-004), 1 página, 2 abas

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` ainda é o modelo não preenchido (placeholders), portanto não define princípios nem gates. **Resultado: sem gates aplicáveis, nenhuma violação.** Reavaliado após o desenho (Fase 1): sem alteração.

Convenções do repositório observadas (CLAUDE.md raiz): texto, comentários e docs em português; módulo independente com `package.json` próprio; sem arquivos fora de `front/`.

## Project Structure

### Documentation (this feature)

```text
specs/001-home-filmes-cinemas/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── catalogo-consumido.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (repository root: `ingressos-golang/front/`)

```text
vite.config.ts                 # + proxy /catalogo → localhost:8082 e bloco test
package.json                   # + vitest, jsdom, @testing-library/*
src/
├── main.tsx
├── index.css                  # estilos globais/tokens da página
├── App.tsx                    # renderiza <PaginaInicial />
├── api/
│   ├── catalogo.ts            # buscarFilmes(), buscarCinemas(): percorre páginas
│   └── tipos.ts               # Filme, Cinema, PaginaApi
├── dominio/
│   ├── texto.ts               # normalizar() sem acento/caixa
│   ├── filtrosFilmes.ts       # filtrarFilmes(), opcoes derivadas
│   └── filtrosCinemas.ts      # filtrarCinemas(), opcoes derivadas
├── hooks/
│   ├── useLista.ts            # estados carregando/erro/pronto + recarregar
│   └── useAbaNaUrl.ts         # sincroniza ?aba= com history
├── componentes/
│   ├── Abas.tsx               # tablist acessível
│   ├── AbaFilmes.tsx          # filtros + lista
│   ├── AbaCinemas.tsx
│   ├── CartaoFilme.tsx
│   ├── CartaoCinema.tsx
│   ├── EstadoLista.tsx        # carregando / vazio / sem resultado / erro
│   └── PaginaInicial.tsx
tests/ (ou colocalizados *.test.ts[x] como no admin)
```

**Structure Decision**: SPA única em `front/src` com camadas simples (`api` → `dominio` puro → `hooks`/`componentes`), espelhando a separação do `admin/` (api isolada, lógica pura testável) sem importar nada dele (módulos independentes). Testes colocalizados `*.test.ts[x]`.

## Complexity Tracking

Sem violações; nada a justificar.
