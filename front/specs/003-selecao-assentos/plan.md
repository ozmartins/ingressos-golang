# Implementation Plan: Página de seleção de assentos ("Comprar Ingressos")

**Branch**: `003-selecao-assentos` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-selecao-assentos/spec.md`

## Summary

O botão "Comprar Ingressos" (card do filme em `PaginaCinema`) passa a navegar para a rota `/cinemas/{cinemaId}/filmes/{filmeId}/comprar` (sessão escolhida em `?sessao={id}`). A rota exige login: sem sessão válida, o app guarda o destino com `urlAutenticacao('login', retorno)` (mecanismo Keycloak/PKCE já existente) e redireciona; ao voltar, `useSessao` já executa `navegar(retorno)`. A página carrega, em paralelo, cinema, filmes e sessões do `catalogo` (público) e o mapa de poltronas do `estoque` (`GET /sessoes/{id}/poltronas`, com Bearer). Mostra cabeçalho da sessão, seletor de sessão (mesmo filme+cinema, futuras), mapa por fileira, legenda, alternância de números e painel "Resumo do pedido". "Continuar" chama `POST /sessoes/{id}/bloqueios` e exibe confirmação com o prazo (`expira_em`); 409 atualiza o mapa e mantém os demais assentos escolhidos. Estilo: tokens já existentes em `index.css` (`--superficie`, `--destaque`, `--borda`…), sem cores novas fora dos tokens. Sem novas dependências.

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: nenhuma nova; Vite 8 — novo proxy `/estoque` → `http://localhost:8085` (estoque não tem CORS)

**Storage**: N/A no servidor; seleção de assentos em estado do React; sessão de login em `sessionStorage` (já existente)

**Testing**: Vitest + Testing Library + jsdom; lógica pura em `dominio/` (mapa, totais, gate de login) e componentes em testes colocalizados; `oxlint`

**Target Platform**: Navegadores modernos, desktop e celular (≥ 360 px)

**Project Type**: Web application (SPA consumindo serviços existentes: `catalogo` e `estoque`)

**Performance Goals**: página utilizável em ≤ 3 s (SC-002); consultas ao catálogo e ao estoque disparadas em paralelo

**Constraints**: JWT Keycloak (aud `cinema-app`) exigido pelo estoque; limite de assentos por pedido = 10 (`POLTRONAS_MAX_POR_BLOQUEIO`, confirmado também pelo 400 com `metadata.limite`); reserva expira em 10 min; preço como decimal em texto → cálculos em centavos inteiros; taxa sem fonte na plataforma → R$ 0,00; sem mudanças no backend

**Scale/Scope**: salas com dezenas a ~200 poltronas; 1 página nova, 1 rota nova, 1 proxy

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações apenas dentro de `front/`; nenhuma mudança no backend; commits direto na `master` (memória do projeto).

## Project Structure

### Documentation (this feature)

```text
specs/003-selecao-assentos/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── rota-compra.md
│   └── estoque-consumido.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
vite.config.ts                       # + proxy '/estoque' -> localhost:8085 (dev e preview)
src/
├── App.tsx                          # + rota 'compra' (com guarda de login)
├── api/
│   ├── tipos.ts                     # + Poltrona, MapaPoltronas, RespostaBloqueio
│   └── estoque.ts                   # NOVO: buscarMapa(sessaoId, token), bloquearPoltronas(...), erros tipados
├── dominio/
│   ├── rotas.ts                     # + Rota 'compra', caminhoCompra(), lerRota()
│   ├── mapaAssentos.ts              # NOVO: agrupar por fileira, ordenar, situação visual
│   ├── pedido.ts                    # NOVO: totais em centavos, limite, alternar seleção
│   └── sessoesDoFilme.ts            # NOVO: sessões futuras do filme no cinema, sessão padrão
├── componentes/
│   ├── ProgramacaoFilme.tsx         # botão passa a navegar (onComprar recebe filme)
│   ├── PaginaCinema.tsx             # liga onComprar -> navegar(caminhoCompra(...))
│   ├── PaginaCompra.tsx             # NOVO: orquestra carga, guarda de login, estados
│   ├── CabecalhoSessao.tsx          # NOVO: cinema/sala/dia/hora + etapa + seletor de sessão
│   ├── MapaAssentos.tsx             # NOVO: grade, tela, legenda, alternar números
│   └── ResumoPedido.tsx             # NOVO: pôster, dados, itens, taxa, total, Continuar
├── hooks/useSessao.ts               # reuso (sem mudança de contrato)
└── index.css                        # estilos novos usando tokens existentes
```

**Structure Decision**: segue o layout do front existente (`api/`, `dominio/` puro, `componentes/`, `hooks/`, testes colocalizados). Sem router externo: estende `rotas.ts` e `App.tsx`.

## Complexity Tracking

Sem violações a justificar.
