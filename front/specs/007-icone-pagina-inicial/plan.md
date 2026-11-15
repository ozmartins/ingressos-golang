# Implementation Plan: Ícone para retornar à página inicial em todas as páginas

**Branch**: `007-icone-pagina-inicial` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-icone-pagina-inicial/spec.md`

## Summary

`App` já renderiza `BarraConta` acima de qualquer rota. Esse é o ponto único para atender FR-001/FR-009: um novo componente `LinkInicio` (link com ícone SVG de casa, `href="/"`, nome acessível "Página inicial") é colocado no mesmo cabeçalho, à esquerda, com a `BarraConta` à direita. Um novo wrapper `CabecalhoSite` em `App` agrupa os dois (linha flex, mesmo espaçamento atual). O clique usa `interceptarClique` + `navegar('/')` (navegação interna sem recarregar; Ctrl/Cmd/botão do meio seguem o padrão do navegador). Já em `/`, o clique é interceptado e nada é feito (sem entrada duplicada no histórico, estado da página inicial preservado). Nenhuma página é alterada: como o cabeçalho vive em `App`, páginas futuras herdam o ícone. Sem dependências novas, sem back-end, sem rede.

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: nenhuma nova (ícone em SVG inline)

**Storage**: nenhum

**Testing**: Vitest + Testing Library + jsdom; teste do `LinkInicio` (nome acessível, destino `/`, navegação interna, Ctrl+clique, no-op na raiz) e teste em `App.test.tsx` de que o ícone aparece em cada rota (inicial, cinema, filme, compra, pagamento) e em estado de erro/carregamento

**Target Platform**: Navegadores modernos, desktop e celular

**Project Type**: Web application (SPA; só toca o front)

**Performance Goals**: sem impacto perceptível (SVG inline de poucos bytes)

**Constraints**: alvo de toque ≥ 44×44 px; foco visível (reaproveita o `outline` com `--foco`); cores só por variáveis CSS existentes (contraste nos temas claro e escuro); não sobrepor `BarraConta` em telas estreitas

**Scale/Scope**: 1 componente novo, 1 wrapper em `App`, estilos mínimos

## Constitution Check

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações só em `front/`; backend intocado; commits direto na `master`.

## Project Structure

### Documentation (this feature)

```text
specs/007-icone-pagina-inicial/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── link-inicio.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
src/
├── componentes/
│   ├── LinkInicio.tsx        # NOVO: link com ícone de casa para "/"
│   ├── LinkInicio.test.tsx   # NOVO
│   ├── CabecalhoSite.tsx     # NOVO: LinkInicio à esquerda + BarraConta à direita
├── App.tsx                   # troca <BarraConta/> por <CabecalhoSite/>
├── App.test.tsx              # + ícone presente em todas as rotas
└── index.css                 # estilos do cabeçalho e do link (ajusta .barra-conta)
```

**Structure Decision**: componentes de apresentação em `src/componentes/`, como os demais; o cabeçalho único em `App` garante cobertura de todas as rotas sem tocar nas páginas.

## Complexity Tracking

Sem desvios.
