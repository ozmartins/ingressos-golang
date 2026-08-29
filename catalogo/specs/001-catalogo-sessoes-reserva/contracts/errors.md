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
| `poltronas-indisponiveis` | 409 | Estoque respondeu `sucesso=false` | FR-026 |
| `estoque-indisponivel` | 503 | Timeout de 2s, `Unavailable` do gRPC, ou recusa rápida com o disjuntor aberto | FR-028, FR-030 |
| `resposta-invalida-do-parceiro` | 502 | Estoque respondeu `sucesso=true` sem `reserva_id` ou sem `expira_em` | Edge case da spec |
| `erro-interno` | 500 | Falha não prevista; `detail` genérico, sem vazar interno | FR-028 |

## Regras

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
