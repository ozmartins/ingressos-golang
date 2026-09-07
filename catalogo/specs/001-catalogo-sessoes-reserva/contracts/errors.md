# Catálogo de erros — RFC 9457

**Feature**: `001-catalogo-sessoes-reserva`

Toda resposta de erro usa `Content-Type: application/problem+json`. O campo `type` é o
contrato estável que o cliente inspeciona (SC-009); `title` e `detail` são texto humano e
podem mudar de redação sem quebrar cliente algum.

Prefixo dos URIs: `https://cinema.example/errors/`

| `type` (sufixo) | Status | Quando | Origem |
|---|---|---|---|
| `parametro-invalido` | 400 | `page` < 1, `page_size` fora de 1..máximo, `status` desconhecido, `data` fora de YYYY-MM-DD, UUID malformado | FR-002, FR-009, FR-018 |
| `corpo-invalido` | 400 | `poltronas_ids` ausente, vazia ou com duplicatas | FR-023 |
| `nao-autenticado` | 401 | Token ausente, malformado, assinatura inválida, expirado, emissor ou audiência não reconhecidos, ou sem claim `sub` | FR-019, FR-020, FR-021 |
| `cinema-nao-encontrado` | 404 | `cinema_id` inexistente na consulta de salas | FR-013 |
| `sessao-nao-encontrada` | 404 | `sessao_id` inexistente na reserva | FR-022 |
| `sessao-nao-reservavel` | 422 | Sessão existe, mas já iniciou, foi finalizada ou cancelada | FR-022 |
| `conflito` | 409 | Entrada bem-formada que colide com o que já está gravado: número de sala repetido entre as ativas do cinema, sala ocupada no horário pedido, tentativa de mover uma sala de cinema ou uma sessão de sala | Escrita de salas e sessões |
| `poltronas-indisponiveis` | 409 | Estoque respondeu `sucesso=false` | FR-026 |
| `reserva-recusada` | 400 | Estoque recusou a solicitação por entrada inválida: rótulo fora do formato, ou mais poltronas que o limite por reserva (o `detail` traz o limite vigente) | Contrato de erros do estoque, `INVALID_ARGUMENT` |
| `poltrona-inexistente` | 422 | Uma ou mais poltronas informadas não existem na sessão | Contrato de erros do estoque, `POLTRONA_INEXISTENTE` |
| `sessao-sem-poltronas` | 422 | A sessão existe no catálogo, mas o estoque ainda não provisionou a matriz de poltronas dela | Contrato de erros do estoque, `SESSAO_NAO_PROVISIONADA` |
| `estoque-indisponivel` | 503 | Timeout de 2s, `Unavailable` do gRPC, ou recusa rápida com o disjuntor aberto | FR-028, FR-030 |
| `resposta-invalida-do-parceiro` | 502 | Estoque respondeu `sucesso=true` sem `reserva_id` ou sem `expira_em`; ou o estoque respondeu `INTERNAL` — defeito dele, que repetir não resolve | Edge case da spec, e o contrato de erros do estoque |
| `erro-interno` | 500 | Falha não prevista; `detail` genérico, sem vazar interno | FR-028 |

## Regras

- **A culpa fica onde é.** O estoque distingue, no seu contrato de erros, o que
  ele decidiu negar do que nele falhou, e a tradução preserva essa distinção. As
  três categorias de recusa (`reserva-recusada`, `poltrona-inexistente`,
  `sessao-sem-poltronas`) já caíram em `estoque-indisponivel` no passado, e o
  efeito era mandar o cliente tentar de novo quando o defeito estava na
  solicitação dele. A tradução olha a `reason` do `ErrorInfo`, nunca o texto da
  mensagem — o texto é livre para mudar de redação.
- **Recusa não abre o disjuntor.** A recusa rápida existe para poupar parceiro
  doente; entrada inválida repetida não é doença do parceiro, e contá-la tiraria
  a reserva do ar para todos por conta de um cliente só.
- **O 503 é indistinguível para o cliente** entre timeout real e recusa rápida do disjuntor — exigência explícita da spec ("sem diferença perceptível"). A distinção existe apenas nas métricas e nos logs (`estoque.bloqueio.total`, rótulo de desfecho).
- **Nunca vazar detalhe interno**: mensagem do gRPC, endereço do estoque, SQL ou stack trace não entram em `detail` (FR-028). Vão para o log, correlacionados pelo `trace_id`.
- **`instance`** carrega o `trace_id` da requisição, para que quem reporta um problema traga consigo a chave que localiza o rastro (SC-011).
- **Erros de validação múltiplos** usam o array `errors` com `campo` e `mensagem`, para o cliente marcar todos os campos de uma vez em vez de descobrir um por requisição.

## Exemplo

```json
{
  "type": "https://cinema.example/errors/poltronas-indisponiveis",
  "title": "Poltronas indisponíveis",
  "status": 409,
  "detail": "Uma ou mais poltronas selecionadas não estão disponíveis.",
  "instance": "urn:trace:4bf92f3577b34da6a3ce929d0e0e4736"
}
```
