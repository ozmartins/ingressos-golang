# Tasks: Baixar o ingresso em PDF após o pagamento aprovado

**Input**: Design documents from `specs/006-ingresso-pdf/` (plan.md, spec.md, research.md, data-model.md, contracts/botao-ingresso-pdf.md, quickstart.md)

**Tests**: incluídos — convenção do front (Vitest + Testing Library colocalizados em `src/**/*.test.ts(x)`).

**Formato**: `- [ ] T### [P?] [US?] Descrição com caminho`. Caminhos relativos a `ingressos-golang/front/`.

## Phase 1: Setup

- [X] T001 Confirmar base verde: `npm test`, `npm run lint` e `npm run build` e anotar falhas pré-existentes
- [X] T002 Instalar a dependência fixa `npm install --save-exact jspdf@4.2.1` (atualiza `package.json` e `package-lock.json`)

## Phase 2: Foundational (bloqueia as histórias)

- [X] T003 [P] Em `src/dominio/reservaPagamento.ts` adicionar `classificacao?: string` a `ReservaPagamento`; em `lerReserva` aceitar ausência (reserva antiga) e rejeitar valor que não seja string; ajustar `src/dominio/reservaPagamento.test.ts` (com, sem e inválida)
- [X] T004 Em `src/componentes/PaginaCompra.tsx` gravar `classificacao: filme.classificacao_etaria` ao salvar o instantâneo (junto de `salvarReserva`, perto da linha ~269) e ajustar `src/componentes/PaginaCompra.test.tsx` para conferir o campo
- [X] T005 [P] Criar `src/dominio/ingressoPdf.ts` com `codigoPedido(reserva)` (FNV-1a determinístico de `sessaoId|rotulos ordenados|expiraEm`, formato `ING-XXXXXXXX`, ver research D3), `nomeArquivoIngresso(reserva)` (research D7) e o tipo `IngressoPdf`; testes em `src/dominio/ingressoPdf.test.ts` (estabilidade do código, mudança com outros assentos/sessão, nome sem acento/caracteres inválidos, título longo truncado)

**Checkpoint**: dependência instalada, instantâneo com classificação, código e nome do arquivo prontos.

## Phase 3: User Story 1 — Baixar o ingresso em PDF (P1) 🎯 MVP

**Goal**: botão em "Pagamento aprovado" que baixa um PDF com os dados do ingresso.

**Independent Test**: pagar (simulado) e baixar; o PDF contém filme, classificação, cinema, sala, dia/horário, assentos, valores e código (quickstart passos 1–5; spec US1).

- [X] T006 [US1] Em `src/dominio/ingressoPdf.ts` implementar `montarIngressos(reserva)`: um item por assento (ordem fileira/número), textos formatados com `formatarCentavos`/`formatarDiaDe`/`formatarHora`, `resumo` com quantidade, ingressos, taxa e total, texto fixo `aviso` de simulação ("Pagamento simulado: este documento não tem validade para entrada", FR-007), `classificacao` omitida se ausente, sem qualquer campo de cartão; testes em `src/dominio/ingressoPdf.test.ts` (1, 3 e 10 assentos, valores, aviso presente, ausência de classificação)
- [X] T007 [US1] Criar `src/infra/gerarPdf.ts` com `baixarIngressoPdf(ingresso: IngressoPdf)`: `import('jspdf')` dinâmico, A4 retrato, uma página por ingresso (filme, classificação, cinema, sala, dia, horário, assento em destaque, código, aviso de simulação no rodapé) e uma página própria de resumo (quantidade, ingressos, taxa, total, código), `doc.save(nomeArquivo)`; teste `src/infra/gerarPdf.test.ts` confere que o `output('arraybuffer')`/blob começa com `%PDF-` e tem N+1 páginas (N ingressos + resumo), inclui um caso com 10 assentos gerado em tempo folgado (SC-002, limite de 3 s) e afirma que `fetch` não é chamado durante a geração (FR-012) (mock de `save` para não baixar)
- [X] T008 [US1] Em `src/componentes/PaginaPagamento.tsx` adicionar prop injetável `gerarPdf` (padrão `baixarIngressoPdf`), estados `gerandoPdf`/`erroPdf` e, no bloco `aprovado`, o botão "Baixar ingresso (PDF)" (`type="button"`, "Gerando PDF…" com `aria-busy` e desabilitado enquanto gera, erro em `role="alert"` com nova tentativa) conforme `contracts/botao-ingresso-pdf.md`; o botão não existe nos demais estados
- [X] T009 [US1] Testes em `src/componentes/PaginaPagamento.test.tsx`: botão só após aprovado (ausente em formulário, processando, recusado, expirada, sem reserva); clique chama o gerador com os dados do pedido (código, assentos, total) e mantém a confirmação; duplo clique não dispara dois downloads simultâneos; falha mostra mensagem e permite tentar de novo; baixar duas vezes entrega o mesmo código

**Checkpoint**: MVP — baixar o PDF com todos os dados.

## Phase 4: User Story 2 — PDF legível e com aviso de simulação (P2)

**Goal**: documento organizado como ingresso, com aviso de simulação e texto sem corte.

**Independent Test**: abrir o PDF e identificar os dados em segundos, ver o aviso e conferir acentos e textos longos (quickstart passos 3 e 6; spec US2).

- [X] T010 [US2] Em `src/infra/gerarPdf.ts` garantir o aviso de simulação no rodapé de **toda** página (ingressos e resumo) com destaque legível em impressão preto e branco; teste em `src/infra/gerarPdf.test.ts` (aviso em todas as páginas)
- [X] T011 [US2] Em `src/infra/gerarPdf.ts` refinar o layout: cabeçalho "Ingressos", assento em fonte grande, código visível, quebra de linha com `splitTextToSize` para filme/cinema e redução de fonte em dois níveis para títulos longos (research D8), acentos via fonte padrão (WinAnsi); testes em `src/infra/gerarPdf.test.ts` com título/cinema muito longos e texto acentuado (`ç ã é ª`), conferindo que o documento é gerado sem corte
- [X] T012 [P] [US2] Estilos mínimos em `src/index.css` para o botão/erro dentro de `.reserva-confirmada` (reutilizar classes existentes; sem rolagem horizontal em 360 px)

## Phase 5: Polish & transversais

- [X] T013 Rodar `npm test`, `npm run lint`, `npm run build`; confirmar no build que o `jspdf` sai em chunk separado do bundle inicial (comparar tamanhos antes/depois)
- [X] T014 Executar o roteiro de `specs/006-ingresso-pdf/quickstart.md` contra a plataforma local (incluindo abrir o PDF e conferir os dados) e registrar desvios
- [X] T015 [P] Atualizar `README.md` do front com o botão de baixar o ingresso em PDF e a dependência `jspdf`

## Dependências e ordem

- Phase 1 → Phase 2 (T003 ‖ T005; T004 depende de T003) → US1 → US2 → Polish.
- Dentro da US1: T006 depende de T003 e T005; T007 de T002, T005 e T006; T008 de T007; T009 de T008.
- US2 edita os mesmos arquivos de T006–T007 (`gerarPdf.ts`); executar após a US1. O aviso de simulação (FR-007) já nasce na US1 (T006/T007), de modo que o MVP seja conforme à spec.

## Exemplo de paralelismo

```text
Foundational:  T003 ‖ T005   (depois T004)
US2:           T012 ‖ (T010 → T011)
Polish:        T015 ‖ (T013 → T014)
```

## Estratégia

1. **MVP** = Setup + Foundational + US1: o PDF baixa com todos os dados do pedido.
2. Incrementar com a US2 (aviso, layout, textos longos) e validar com o quickstart.
3. Fechar com Polish (testes, lint, build, chunk separado, README).
