# Implementation Plan: Página de detalhes do cinema (salas, sessões e filmes)

**Branch**: `002-detalhe-cinema` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-detalhe-cinema/spec.md`

## Summary

O card de cinema passa a ser um link para `/cinemas/{id}`. A nova página consulta o `catalogo` (somente leitura) por cinema: `GET /cinemas/{id}`, `GET /salas?cinema_id=&ativo=true`, `GET /sessoes?cinema_id=` (a grade já exclui canceladas e finalizadas e traz título do filme, número e tipo da sala) e `GET /filmes` para enriquecer cada filme (gênero, duração, classificação, imagem). A montagem (agrupar por filme, ordenar, filtrar por dia, ligar sala↔sessão) é feita por funções puras no navegador. O roteamento é mínimo, sem biblioteca: o `App` lê `location.pathname`, e a `PaginaInicial` permanece montada (oculta) enquanto o detalhe é exibido, de modo que filtros, aba e dados sobrevivem ao "voltar".

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: nenhuma nova (sem router, sem gerenciador de dados); Vite 8 com proxy `/catalogo` já configurado

**Storage**: N/A (somente leitura; filtro de dia em memória)

**Testing**: Vitest + Testing Library + jsdom (já configurados); lógica pura em `dominio/`, componentes e roteamento em testes colocalizados `*.test.ts[x]`; `oxlint`

**Target Platform**: Navegadores modernos, desktop e celular (≥ 360 px)

**Project Type**: Web application (SPA frontend consumindo serviço existente)

**Performance Goals**: conteúdo (ou carregamento seguido do resultado) em ≤ 3 s (SC-003); as 4 consultas disparadas em paralelo

**Constraints**: catálogo sem CORS (proxy); `page_size` máximo 100 (reusa `buscarTodas`); sem login; datas exibidas no fuso local do navegador; preço vem como decimal em texto (`"42.00"`)

**Scale/Scope**: dezenas de salas e centenas de sessões por cinema; 1 página nova, 1 rota

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações apenas dentro de `front/`; nenhuma mudança no backend.

## Project Structure

### Documentation (this feature)

```text
specs/002-detalhe-cinema/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── catalogo-consumido.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
src/
├── App.tsx                         # escolhe a rota; mantém PaginaInicial montada
├── api/
│   ├── tipos.ts                    # + Sala, SessaoNaGrade, TipoTela, Idioma
│   └── catalogo.ts                 # + buscarCinema(id), buscarSalas(cinemaId), buscarSessoes(cinemaId)
├── dominio/
│   ├── programacao.ts              # montarProgramacao(): agrupa por filme, ordena, filtra por dia, salas sem sessão
│   ├── formatar.ts                 # + formatarPreco(), formatarDataHora(), formatarDia()
│   └── rotas.ts                    # caminhoCinema(id), lerRota(pathname)
├── hooks/
│   ├── useRota.ts                  # pathname + navegar() com history/popstate
│   └── useRecurso.ts               # carga única (estado/erro/recarregar) para um valor
└── componentes/
    ├── CartaoCinema.tsx            # passa a ser <a href> (link) com navegação sem recarga
    ├── PaginaCinema.tsx            # cabeçalho, salas, filtro de dia, programação
    ├── CabecalhoCinema.tsx
    ├── ListaSalas.tsx
    └── ProgramacaoFilme.tsx        # filme + sessões (hora, sala, tipo, idioma, preço)
vite.config.ts                      # nada a alterar (SPA fallback do Vite já cobre /cinemas/:id)
```

**Structure Decision**: mantém as camadas da feature 001 (`api` → `dominio` puro → `hooks`/`componentes`). Roteamento por `pathname` com `history.pushState` (mesma técnica do `useAbaNaUrl`), evitando nova dependência para uma única rota.

## Complexity Tracking

Sem violações; nada a justificar.
