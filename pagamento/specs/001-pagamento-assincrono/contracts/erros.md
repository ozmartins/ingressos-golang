# Contrato de Erros — Servico-Pagamento

Os erros seguem a RFC 9457 (`application/problem+json`). O que é contrato é o par
**status HTTP + `type`** (`https://cinema.example/errors/<codigo>`; a coluna `codigo`
abaixo é o último segmento). `title` e `detail` são texto humano e podem mudar de redação sem versão nova. Nenhum erro expõe detalhe interno: mensagem
de driver, SQL, rastro de pilha ou identificador de linha nunca chegam à resposta.

## Categorias da API de consulta

| Status | `codigo` | Quando | Requisito |
|---|---|---|---|
| 400 | `reserva-id-invalido` | `reserva_id` não é UUID válido | FR-018 |
| 401 | `credencial-invalida` | token ausente, malformado, expirado, assinatura ou emissor inválidos | FR-016 |
| 404 | `pagamento-nao-encontrado` | não há transação para a reserva **ou** a transação é de outra pessoa | FR-017, FR-018 |
| 503 | `servico-indisponivel` | armazenamento inacessível | FR-018 |

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
