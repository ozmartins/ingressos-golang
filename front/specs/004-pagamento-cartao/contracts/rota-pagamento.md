# Contrato de UI: rota de pagamento

**Endereço**: `/cinemas/{cinemaId}/filmes/{filmeId}/pagamento` (`caminhoPagamento(cinemaId, filmeId)`; interpretado por `lerRota` como `{ tipo: 'pagamento', cinemaId, filmeId }`).

**Entrada**: instantâneo `ReservaPagamento` em `sessionStorage` (ver `data-model.md`), gravado por `PaginaCompra` quando a reserva é criada.

| Situação | Comportamento |
|----------|---------------|
| Sem login | `useGuardaLogin` → Keycloak com retorno ao endereço atual; com instantâneo válido na aba a página abre; sem ele cai em "sem reserva" |
| Sem instantâneo válido (ou de outro usuário) | mensagem "Nenhuma reserva em andamento" + links: escolher assentos (`caminhoCompra`) e página inicial |
| Reserva vencida | "Reserva expirada"; formulário desabilitado; link para escolher assentos |
| Reserva válida | resumo + contagem + formulário |
| Aprovado | "Pagamento aprovado" + resumo + link para início; instantâneo removido |
| Recusado (número terminado em `0000`) | "Pagamento recusado" + botão "Tentar outro cartão" |

**Voltar**: link "Voltar" → `caminhoCompra(cinemaId, filmeId, sessaoId)`.

**Rede**: nenhuma chamada nova. A única chamada relevante (`POST /sessoes/{id}/bloqueios`) pertence à feature 003 e permanece inalterada.

**Acessibilidade**: cada campo com `<label>`, erro ligado por `aria-describedby`/`aria-invalid`; resultado em região `role="status"` (aprovado) ou `role="alert"` (recusado/erro); contagem com `role="timer"`.
