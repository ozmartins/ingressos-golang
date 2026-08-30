# Catálogo de Erros — Servico-Estoque

Fonte: FR-046 (categorias distinguíveis sem interpretar texto) e o princípio
"Erro é Contrato". A **categoria é o código de status gRPC**; a mensagem humana
é livre para mudar de redação e NÃO deve ser comparada por nenhum cliente.

## Regra estrutural

`RespostaBloqueio.sucesso = false` significa **exclusivamente** "alguma poltrona
solicitada não está livre" (com `motivo = POLTRONAS_INDISPONIVEIS`). Qualquer
outra falha é sinalizada por status gRPC de erro, sem corpo de resposta.

Isso existe porque o `Servico-Catalogo` já traduz `sucesso=false` para HTTP 409
`poltronas-indisponiveis`; usar `sucesso=false` para entrada inválida faria o
catálogo devolver 409 para um erro do cliente.

## Mapa de status

| Status gRPC | Quando | Requisito | `ErrorInfo.reason` |
|---|---|---|---|
| `OK` + `sucesso=true` | Bloqueio concedido | FR-001, FR-007 | — |
| `OK` + `sucesso=false` | Ao menos uma poltrona RESERVADA ou OCUPADA | FR-002, FR-003 | — |
| `INVALID_ARGUMENT` | Lista vazia, rótulos repetidos, `usuario_id` ausente, rótulo fora do formato, acima do limite por bloqueio | FR-003, FR-004 | `SOLICITACAO_INVALIDA`, `LIMITE_POLTRONAS_EXCEDIDO` |
| `FAILED_PRECONDITION` | Sessão sem matriz provisionada, ou rótulo inexistente na sessão | FR-003, FR-036 | `SESSAO_NAO_PROVISIONADA`, `POLTRONA_INEXISTENTE` |
| `NOT_FOUND` | `ConsultarMapaPoltronas` para sessão desconhecida | FR-031 | `SESSAO_DESCONHECIDA` |
| `UNAVAILABLE` | Banco, mecanismo de exclusividade ou dependência de estado indisponível — nenhum estado foi alterado | FR-006, SC-012 | `DEPENDENCIA_INDISPONIVEL` |
| `DEADLINE_EXCEEDED` | Orçamento da operação estourado antes de decidir | SC-001 | — |
| `INTERNAL` | Falha que o catálogo de erros de domínio não prevê — defeito deste serviço, não do chamador. A resposta é sempre a mesma, sem metadados: o detalhe fica só no log, correlacionado pelo `trace_id` | — | `ERRO_INTERNO` |
| `UNAUTHENTICATED` / recusa no handshake | Chamador sem identidade de serviço válida | FR-037 | — (a conexão TLS nem se estabelece) |

`LIMITE_POLTRONAS_EXCEDIDO` acompanha o limite vigente em `ErrorInfo.metadata["limite"]`,
para que o chamador possa informar a pessoa usuária sem consultar documentação (FR-004).

`ERRO_INTERNO` é a categoria de escape: qualquer erro que não case com um erro de
domínio conhecido chega ao cliente por ela. Repetir uma resposta `INTERNAL` não
tem por que dar certo — quem integra deve tratá-la como defeito a reportar, não
como falha transitória a retentar (esta última é `UNAVAILABLE`).

## Detalhes tipados

Erros carregam `google.rpc.ErrorInfo` com `reason` (tabela acima) e
`domain = "estoque.ingressos"`. A `reason` é parte do contrato e só muda com
versão nova; a mensagem livre, não.

## O que nunca aparece em resposta de erro

Mensagem do driver de banco, SQL, endereço de dependência, nome de fila,
pilha de execução ou material criptográfico. Tudo isso vai para o log
estruturado, correlacionado pelo `trace_id` propagado (FR-044, FR-039).

## Nota de integração

O adaptador atual do `Servico-Catalogo` mapeia `Unavailable` → HTTP 503 e trata
o restante como falha genérica. Para que `INVALID_ARGUMENT` e
`FAILED_PRECONDITION` cheguem ao cliente como 400/422 em vez de 5xx, o catálogo
precisa estender seu mapeamento. Enquanto não o fizer, o comportamento é seguro
(erro do cliente vira erro genérico), apenas menos informativo.
