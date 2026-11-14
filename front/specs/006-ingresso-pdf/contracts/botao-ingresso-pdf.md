# Contrato de UI: botão "Baixar ingresso (PDF)"

**Local**: dentro do bloco "Pagamento aprovado" de `PaginaPagamento` (estado `aprovado`), abaixo do texto de confirmação e antes do link "Voltar à página inicial".

| Situação | Comportamento |
|----------|---------------|
| Formulário, processando, recusado, reserva expirada, sem reserva | botão **não** é renderizado |
| Aprovado | botão `type="button"` "Baixar ingresso (PDF)" habilitado |
| Gerando | botão desabilitado com texto "Gerando PDF…" (`aria-busy`) |
| Sucesso | download do arquivo; a confirmação permanece |
| Falha | `role="alert"` "Não foi possível gerar o PDF." + botão volta a ficar habilitado para tentar de novo |

**Arquivo**: `ingresso-{filme}-{dd-mm}-{hhmm}.pdf`, A4 retrato, uma página por assento + uma página própria de resumo de valores (N+1 páginas).

**Conteúdo por página**: marca "Ingressos", filme, classificação (se houver), cinema, sala, dia e horário, assento em destaque, código do pedido; rodapé com o aviso de simulação. **Resumo**: quantidade, valor dos ingressos, taxa, total, código do pedido.

**Proibido no documento**: qualquer dado de cartão.

**Rede**: nenhuma chamada ao gerar (a biblioteca vem do chunk do próprio app).

**Acessibilidade**: botão focável e acionável por teclado; estado de andamento anunciado por `aria-busy`/texto; erro em região `role="alert"`.
