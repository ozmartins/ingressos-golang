# Implementation Plan: Baixar o ingresso em PDF após o pagamento aprovado

**Branch**: `006-ingresso-pdf` (sem branch dedicada; trabalho direto na `master`) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/006-ingresso-pdf/spec.md`

## Summary

Em `PaginaPagamento`, o estado `aprovado` passa a mostrar o botão "Baixar ingresso (PDF)". O clique chama `baixarIngressoPdf(reserva)`, que carrega a biblioteca **jsPDF** sob demanda (`import()` dinâmico, chunk separado: nada muda no carregamento inicial), monta o documento A4 retrato com uma página por assento (dados da sessão, assento em destaque, código do pedido, aviso de simulação) e uma seção de resumo de valores, e dispara o download pelo navegador. Toda a montagem do conteúdo é uma função pura em `dominio/ingressoPdf.ts` (`montarIngressos(reserva)` → estrutura de textos já formatados, `codigoPedido(reserva)`, `nomeArquivoIngresso(reserva)`); só o adaptador `infra/gerarPdf.ts` conhece o jsPDF, o que mantém a regra testável sem PDF. O instantâneo da reserva ganha o campo `classificacao` (gravado em `PaginaCompra`, lido de forma tolerante em `lerReserva`) para cumprir o FR-004. Tudo local: sem rede, sem backend, sem dados de cartão. Uma nova dependência (`jspdf`) fixada em versão exata.

## Technical Context

**Language/Version**: TypeScript ~7.0, React 19.3 (existentes)

**Primary Dependencies**: `jspdf` 4.2.1 (nova, carregada por `import()` dinâmico). Alternativas em `research.md`.

**Storage**: nenhum novo; usa o instantâneo `pagamento:reserva` já existente (apenas 1 campo a mais, sem dados de cartão)

**Testing**: Vitest + Testing Library + jsdom; regra pura (`ingressoPdf`) testada diretamente; componente testado com o gerador injetável (`gerarPdf`), como `irParaLogin`/`atrasoMs`; um teste do adaptador confere que o arquivo produzido é um PDF (`%PDF-`) com o texto esperado

**Target Platform**: Navegadores modernos, desktop e celular

**Project Type**: Web application (SPA; só toca o front)

**Performance Goals**: download em ≤ 3 s para 10 assentos; biblioteca fora do bundle inicial

**Constraints**: sem chamadas de rede ao gerar; fontes padrão do PDF (Helvetica, codificação WinAnsi cobre acentos do português); sem imagens remotas (pôster omitido: evita CORS e arquivo pesado); nenhum dado de cartão no documento

**Scale/Scope**: 1 botão, 1 módulo de domínio, 1 adaptador, ajuste do instantâneo e de `PaginaCompra`

## Constitution Check

`.specify/memory/constitution.md` continua sendo o modelo não preenchido; não há princípios nem gates. **Resultado: sem violações.** Reavaliado após o desenho: sem alteração.

Convenções observadas (CLAUDE.md raiz): textos, comentários e docs em português; alterações só em `front/`; backend intocado; commits direto na `master`. Nova dependência justificada em `research.md` (D1).

## Project Structure

### Documentation (this feature)

```text
specs/006-ingresso-pdf/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── botao-ingresso-pdf.md
├── checklists/requirements.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (`ingressos-golang/front/`)

```text
package.json                         # + "jspdf": "4.2.1"
src/
├── dominio/
│   ├── reservaPagamento.ts          # + campo classificacao (opcional na leitura)
│   └── ingressoPdf.ts               # NOVO: codigoPedido, nomeArquivoIngresso, montarIngressos
├── infra/
│   └── gerarPdf.ts                  # NOVO: renderiza a estrutura com jsPDF (import dinâmico) e baixa
├── componentes/
│   ├── PaginaCompra.tsx             # grava classificacao no instantâneo
│   └── PaginaPagamento.tsx          # botão "Baixar ingresso (PDF)" no estado aprovado, erro/andamento
└── index.css                        # estilos mínimos (ação e mensagem de erro)
```

**Structure Decision**: domínio puro continua em `dominio/`; o único código que toca a biblioteca fica isolado em `src/infra/gerarPdf.ts` (novo diretório pequeno, justificado pela separação regra/adaptador já usada nos serviços do repositório).

## Complexity Tracking

| Desvio | Por quê | Alternativa mais simples rejeitada |
|--------|---------|------------------------------------|
| Nova dependência `jspdf` | gerar PDF com texto, paginação e acentos corretos | PDF escrito à mão: frágil (xref, codificação, quebra de linha) e difícil de manter |
