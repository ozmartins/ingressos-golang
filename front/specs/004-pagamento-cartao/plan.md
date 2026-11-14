# Implementation Plan: Página de pagamento via cartão (simulada)

**Branch**: `004-pagamento-cartao` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-pagamento-cartao/spec.md`

## Summary

Em `PaginaCompra`, quando `c.reservar(...)` resolve, o app grava um "instantâneo da reserva" (filme, cinema, sala, horário, assentos, total em centavos, `expira_em`) em `sessionStorage` e navega para a nova rota `/cinemas/{cinemaId}/filmes/{filmeId}/pagamento`. A rota exige login (mesmo `useGuardaLogin`) e uma reserva válida no instantâneo; sem ela, mostra o estado "sem reserva" com links; com prazo vencido, mostra "reserva expirada" e bloqueia o pagamento. A página `PaginaPagamento` mostra o resumo, a contagem regressiva e um formulário de cartão (número, nome, validade, CVV) com máscara e validação em `dominio/cartao.ts` (validade e CVV; o número é livre). "Pagar" é 100% local: `setTimeout` ~2 s simulando processamento, depois aprovado — ou recusado se o número termina em `0000`. Nenhum dado de cartão sai do componente (estado React apenas; sem rede, sem storage, sem log). Na aprovação o instantâneo é removido. Estilo: tokens de `index.css`. Sem backend, sem novas dependências.

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: nenhuma nova

**Storage**: `sessionStorage` apenas para o instantâneo da reserva (sem dados de cartão); nada no servidor

**Testing**: Vitest + Testing Library + jsdom; lógica pura (`cartao`, `reservaPagamento`) em `dominio/`, componente em teste colocalizado; `oxlint`; relógio/atraso injetáveis para testes (fake timers)

**Target Platform**: Navegadores modernos, desktop e celular (≥ 360 px)

**Project Type**: Web application (SPA; esta feature só toca o front)

**Performance Goals**: página interativa imediatamente (sem chamadas de rede); processamento simulado ~2 s

**Constraints**: pagamento simulado, sem chamadas ao serviço `pagamento` nem eventos; valores em centavos inteiros (reuso de `calcularPedido`/formatação existentes); expiração calculada pelo relógio do cliente a partir de `expira_em`

**Scale/Scope**: 1 página nova, 1 rota nova, 1 módulo de domínio de cartão, 1 módulo de instantâneo, pequena alteração em `PaginaCompra`

## Constitution Check

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações só em `front/`; backend intocado; commits direto na `master`.

## Project Structure

### Documentation (this feature)

```text
specs/004-pagamento-cartao/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── rota-pagamento.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
src/
├── App.tsx                          # + rota 'pagamento'
├── dominio/
│   ├── rotas.ts                     # + Rota 'pagamento', caminhoPagamento(), lerRota()
│   ├── cartao.ts                    # NOVO: máscaras, validade, CVV, validarCartao(), cartaoRecusado()
│   └── reservaPagamento.ts          # NOVO: salvar/ler/limpar instantâneo; segundosRestantes()
├── componentes/
│   ├── PaginaCompra.tsx             # após reservar: salva instantâneo e navega ao pagamento
│   ├── PaginaPagamento.tsx          # NOVO: guarda de login, estados, resumo, contagem, formulário, resultado
│   └── FormularioCartao.tsx         # NOVO: campos com máscara/erros acessíveis
└── index.css                        # estilos novos usando tokens existentes
```

**Structure Decision**: mesmo layout do front (domínio puro, componentes, hooks, testes colocalizados); sem router externo.

## Complexity Tracking

Sem violações a justificar.
