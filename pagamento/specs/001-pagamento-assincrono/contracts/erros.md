# Contrato de Erros — Servico-Pagamento

O que é contrato é o par **status HTTP + `codigo`**. A `mensagem` é texto humano e
pode mudar de redação sem versão nova. Nenhum erro expõe detalhe interno: mensagem
de driver, SQL, rastro de pilha ou identificador de linha nunca chegam à resposta.

## Categorias da API de consulta

| Status | `codigo` | Quando | Requisito |
|---|---|---|---|
| 400 | `RESERVA_ID_INVALIDO` | `reserva_id` não é UUID válido | FR-018 |
| 401 | `CREDENCIAL_INVALIDA` | token ausente, malformado, expirado, assinatura ou emissor inválidos | FR-016 |
| 404 | `PAGAMENTO_NAO_ENCONTRADO` | não há transação para a reserva **ou** a transação é de outra pessoa | FR-017, FR-018 |
| 503 | `SERVICO_INDISPONIVEL` | armazenamento inacessível | FR-018 |

**A colisão de 404 é deliberada e é requisito, não simplificação.** Responder 403
para reserva de terceiro confirmaria que ela existe, o que a FR-017 proíbe. As duas
situações devolvem exatamente o mesmo status, o mesmo `codigo` e a mesma mensagem;
o registro interno distingue as duas, a resposta não.

Nenhuma resposta de erro varia conforme a existência de transação alheia — nem em
tamanho, nem em tempo perceptível.

## Categorias do consumo de eventos

O consumo não responde a ninguém; a "resposta" é o destino da mensagem. As
categorias abaixo são o contrato observável pela operação (fila morta e registros).

| Categoria | Destino da mensagem | Estado da transação | Anúncio |
|---|---|---|---|
| Anúncio inválido (campo ausente, valor não positivo, forma desconhecida, JSON quebrado) | fila morta, com motivo em cabeçalho | não é criada | nenhum |
| Reserva já expirada | confirmada | `CANCELADO` / `RESERVA_EXPIRADA` | `pagamento.falhou` |
| Recusa do adquirente | confirmada | `RECUSADO` + motivo | `pagamento.falhou` |
| Ausência de resposta do adquirente | fila morta | `PENDENTE_VERIFICACAO` | **nenhum** |
| Falha transitória (banco ou broker fora) | devolvida à fila | inalterada | nenhum |
| Entregas esgotadas (limite de 3) | fila morta, pelo broker | como estiver | nenhum |

A distinção que mais importa: **inválido** e **recusado** são desfechos diferentes.
O inválido nunca vira transação e nunca é anunciado — sem `reserva_id` válido não
há sequer chave com que gravar. O recusado é uma transação legítima que terminou em
negativa e é anunciada como tal.
