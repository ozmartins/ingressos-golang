# Data Model: Página de pagamento via cartão (simulada)

Tudo vive no cliente; nada é persistido no servidor.

## ReservaPagamento (instantâneo em `sessionStorage`, chave `pagamento:reserva`)

| Campo | Tipo | Observação |
|-------|------|------------|
| `usuarioId` | string \| null | `sub` do access token de quem reservou; deve coincidir com o usuário logado (ambos `null` se o token não for legível) |
| `cinemaId` | string | deve coincidir com a rota |
| `filmeId` | string | deve coincidir com a rota |
| `sessaoId` | string | sessão reservada |
| `filmeTitulo` | string | exibido no resumo |
| `posterUrl` | string \| null | marcador padrão se ausente |
| `cinemaNome` | string | |
| `sala` | string | |
| `inicio` | string (ISO) | dia/horário da sessão |
| `rotulos` | string[] | assentos reservados (ex.: `A5`) |
| `precoCentavos`, `taxaCentavos`, `totalCentavos` | number | inteiros |
| `expiraEm` | string (ISO) | de `expira_em` da resposta do bloqueio |

**Regras**: inválido se JSON malformado, campos faltando, `usuarioId` diferente do usuário logado, `rotulos` vazio, ids diferentes da rota. Expirada se `expiraEm <= agora`. Gravado ao reservar com sucesso; removido na aprovação.

## DadosCartao (somente estado React; nunca persistido)

| Campo | Regra |
|-------|-------|
| `numero` | qualquer número preenchido (sem validação de tamanho ou Luhn); exibido em grupos de 4 |
| `nome` | ≥ 2 caracteres não brancos, acentos aceitos |
| `validade` | `MM/AA`, mês 1–12, não vencida |
| `cvv` | 3–4 dígitos |

## Estado do pagamento

`ocioso → processando → aprovado | recusado`; `recusado → ocioso` (tentar outro cartão, mantém o resumo); `expirada` bloqueia `processando`.
