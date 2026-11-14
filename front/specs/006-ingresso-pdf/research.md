# Research: Ingresso em PDF

## D1 — Biblioteca de PDF
- **Decision**: `jspdf` 4.2.1 (versão exata), carregada com `import('jspdf')` no clique.
- **Rationale**: geração 100% no cliente, fontes padrão (Helvetica) com WinAnsi cobrindo acentos do português, API simples para texto, linhas, retângulos e páginas; o `import()` dinâmico mantém o bundle inicial intacto (o chunk só baixa quando o visitante pede o PDF). Dependências leves (`fflate`, `fast-png`).
- **Alternatives**: `pdf-lib` (1.17.1, parada há anos; texto exige fonte e medição manuais); `pdfmake` (maior, fontes embutidas pesadas); gerar PDF à mão (frágil); `window.print()` com CSS de impressão (não baixa arquivo, depende do diálogo do navegador).

## D2 — Onde fica o código
- **Decision**: regra pura em `dominio/ingressoPdf.ts` devolvendo textos já formatados; adaptador `infra/gerarPdf.ts` só desenha e baixa.
- **Rationale**: testes unitários verificam conteúdo (campos, valores, código, nome do arquivo) sem inspecionar bytes; só um teste do adaptador confirma que sai um PDF.

## D3 — Código do pedido
- **Decision**: `codigoPedido(reserva)` = hash determinístico (FNV-1a 32 bits ×2 → base 36, 8 caracteres maiúsculos) de `sessaoId|rotulos ordenados|expiraEm`, prefixado "ING-" (ex.: `ING-7K3F9Q2A`).
- **Rationale**: o pedido simulado não gera id do servidor (o `reserva_id` do estoque não é guardado no instantâneo); `expiraEm` é fixo por reserva, logo o código é estável entre downloads e distinto entre pedidos. Não é segredo nem prova de validade (documento é simulação).
- **Alternatives**: guardar `reserva_id` no instantâneo (mais fiel, mas exige alterar também `PaginaCompra`/testes da 004 além do necessário e não é um ingresso real); UUID aleatório por clique (viola FR-006).

## D4 — Classificação etária no PDF
- **Decision**: acrescentar `classificacao?: string` ao instantâneo (gravada em `PaginaCompra` a partir de `filme.classificacao_etaria`); `lerReserva` aceita sua ausência (instantâneos antigos na aba) e o PDF omite a linha nesse caso.
- **Rationale**: FR-004 pede classificação e o instantâneo hoje não a tem; leitura tolerante evita invalidar reservas já gravadas.

## D5 — Pôster
- **Decision**: não incluir pôster no PDF.
- **Rationale**: imagem remota exigiria CORS e carga extra; a spec permite o PDF sem imagem (FR-012, sem consulta externa).

## D6 — Estado do botão
- **Decision**: `PaginaPagamento` guarda `gerandoPdf` e `erroPdf`; o botão fica desabilitado enquanto gera, e o erro (`role="alert"`) permite nova tentativa. O gerador é injetável (`gerarPdf` em `Props`) para testes.
- **Rationale**: FR-011 e o edge case de cliques repetidos.

## D7 — Nome do arquivo
- **Decision**: `ingresso-{titulo-sem-acento-em-minusculas}-{dd-mm}-{hhmm}.pdf`; caracteres fora de `[a-z0-9-]` viram `-`, hífens colapsados, título limitado a 40 caracteres.
- **Rationale**: legível e válido em qualquer sistema de arquivos (FR-010).

## D9 — Posição do resumo
- **Decision**: o resumo de valores fica em página própria, após os ingressos (N+1 páginas).
- **Rationale**: evita competir por espaço com o ingresso e fixa a contagem de páginas verificável nos testes.

## D8 — Texto longo
- **Decision**: o adaptador usa `splitTextToSize` do jsPDF para quebrar título/cinema na largura útil; fonte reduzida em dois níveis para títulos muito longos.
- **Rationale**: FR / US2 cenário 3 (sem corte nem sobreposição).
