# Implementation Plan: Cinemas com o filme em cartaz

**Branch**: `005-cinemas-do-filme` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-cinemas-do-filme/spec.md`

## Summary

O card de filme (`CartaoFilme`) passa a ser clicável (mesmo padrão de `CartaoCinema`: link no título + card inteiro clicável) e leva à nova rota `/filmes/{filmeId}`, atendida por `PaginaFilme`. A página carrega o filme (`GET /filmes/{id}`), as sessões do filme em todos os cinemas (`GET /sessoes?filme_id=…`, paginado) e os cinemas ativos (já existente `buscarCinemas`). Uma função pura em `dominio/cinemasDoFilme.ts` cruza as três fontes: mantém sessões futuras, une por `cinema_id` e devolve só cinemas ativos com ao menos uma sessão futura, ordenados por UF/cidade/nome. Os filtros de UF e cidade reusam `filtrosCinemas` (`opcoesCinemas`, `ajustarCidade`, `filtrarCinemas`) sem busca por texto. Cada item (`CartaoCinemaDoFilme`) traz o botão "Comprar Ingressos" que chama `navegar(caminhoCompra(cinema.id, filme.id))` — exatamente a ação do botão de `PaginaCinema`; o login é exigido pelo `useGuardaLogin` já presente em `PaginaCompra`, com retorno à compra. "Voltar" na compra usa `history.back()` quando veio do app, retornando à página do filme. Sem backend novo, sem dependências novas.

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: nenhuma nova

**Storage**: N/A (filtros em estado React; nada persistido)

**Testing**: Vitest + Testing Library + jsdom; lógica pura em `dominio/` e componentes com testes colocalizados; `oxlint`

**Target Platform**: Navegadores modernos, desktop e celular (≥ 360 px)

**Project Type**: Web application (SPA; só toca o front)

**Performance Goals**: listagem interativa logo após as 3 consultas (em paralelo); filtros instantâneos (cliente)

**Constraints**: usar apenas endpoints públicos já existentes do catálogo (`/filmes/{id}`, `/sessoes?filme_id`, `/cinemas?ativo=true`); sem alteração no contrato do catálogo; comportamento do botão idêntico ao da `PaginaCinema`

**Scale/Scope**: 1 página nova, 1 rota nova, 1 módulo de domínio, 1 função de API, 1 componente de item, pequena alteração em `CartaoFilme` e `App`

## Constitution Check

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações só em `front/`; backend intocado; commits direto na `master`.

## Project Structure

### Documentation (this feature)

```text
specs/005-cinemas-do-filme/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── rota-filme.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
src/
├── App.tsx                          # + rota 'filme' → PaginaFilme
├── api/
│   └── catalogo.ts                  # + buscarFilme(id), FilmeNaoEncontrado, buscarSessoesDoFilme(filmeId)
├── dominio/
│   ├── rotas.ts                     # + Rota 'filme', caminhoFilme(), lerRota()
│   └── cinemasDoFilme.ts            # NOVO: cinemasComOFilme(sessoes, cinemas, agora)
├── componentes/
│   ├── CartaoFilme.tsx              # card clicável → caminhoFilme
│   ├── PaginaFilme.tsx              # NOVO: cabeçalho do filme, filtros UF/cidade, estados, lista
│   └── CartaoCinemaDoFilme.tsx      # NOVO: item com dados do cinema + "Comprar Ingressos"
└── index.css                        # estilos mínimos reaproveitando tokens/classes existentes
```

**Structure Decision**: mesmo layout do front (domínio puro, componentes, hooks, testes colocalizados); sem router externo; `PaginaInicial` continua montada e oculta durante a página do filme para preservar aba e filtros (FR-012).

## Complexity Tracking

Sem violações a justificar.
