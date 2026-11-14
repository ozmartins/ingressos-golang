# Quickstart: validar "Ingresso em PDF"

## Pré-requisitos
- `cd ingressos-golang/front && npm install` (instala `jspdf`)
- Plataforma de pé (`docker compose up`) para reservar assentos e acessar o pagamento (ou usar o fluxo simulado com Keycloak local)

## Testes automatizados
```bash
npm test          # Vitest (dominio/ingressoPdf, infra/gerarPdf, PaginaPagamento, PaginaCompra)
npm run lint
npm run build     # confere que o jspdf sai em chunk separado
```

## Validação manual (`npm run dev`)
1. Logado, escolher 1 assento numa sessão → Continuar → pagar com um cartão válido (número que não termine em `0000`).
2. Em "Pagamento aprovado": o botão "Baixar ingresso (PDF)" aparece; clicar baixa `ingresso-<filme>-<dd-mm>-<hhmm>.pdf` sem sair da página.
3. Abrir o PDF: filme, classificação, cinema, sala, dia/horário, assento, resumo (ingressos, taxa, total), código `ING-…` e o aviso de simulação; acentos corretos.
4. Repetir com 3 assentos: 3 páginas de ingresso (um assento cada) e o mesmo código em todas; baixar duas vezes → mesmo código.
5. Cartão terminado em `0000` (recusado), reserva expirada e página sem reserva: sem botão.
6. Título longo: sem corte ou sobreposição. Confirmar que o PDF não contém número/validade/CVV do cartão.
7. Aba Network no DevTools: ao clicar, só aparece o carregamento do chunk do `jspdf` (nenhuma chamada a serviços).

Contrato: [contracts/botao-ingresso-pdf.md](contracts/botao-ingresso-pdf.md) · Modelo: [data-model.md](data-model.md)
