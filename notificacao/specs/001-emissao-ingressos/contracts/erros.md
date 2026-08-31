# Contrato de Erros: Servico-Notificacao

**Fase 1** | **Data**: 2026-08-30

O princípio II exige teste do caminho de sucesso **e de cada categoria de erro
declarada no contrato**. Esta é a lista fechada dessas categorias. Categoria que não
está aqui não existe; categoria que está aqui tem teste.

---

## 1. Duas formas de resposta, e por quê

A ERS fixa o corpo da recusa da portaria como `{"valido": false, "mensagem": "..."}`.
Esse formato é o contrato com o dispositivo de catraca e **não** é RFC 9457.

Erros de protocolo — credencial, corpo malformado, filtro inválido — usam
`application/problem+json` (RFC 9457), como no resto do workspace.

A divisão é por natureza, não por rota: `valido: false` responde à pergunta "posso
deixar entrar?"; `Problema` responde "sua requisição não deu para processar". Uma
catraca sempre recebe uma resposta que sabe interpretar quando a pergunta chegou a ser
feita.

---

## 2. API — `GET /api/v1/ingressos/meus-ingressos`

| Situação | Status | Corpo | Requisito | Teste |
|---|---|---|---|---|
| Pessoa com ingressos | 200 | array ordenado do mais recente ao mais antigo | FR-013, FR-023 | contrato |
| Pessoa sem ingressos | 200 | `[]` — nunca 404 | US3 cenário 5 | contrato |
| Filtro por estado válido | 200 | array recortado, mesma ordem | FR-024 | contrato |
| Filtro por estado desconhecido | 400 | `Problema` | FR-024 | contrato |
| `Authorization` ausente | 401 | `Problema` | FR-015 | contrato |
| Token expirado | 401 | `Problema` | FR-015 | contrato |
| Assinatura, emissor ou público inválidos | 401 | `Problema` | FR-015 | contrato |
| Apresenta `X-API-Key` em vez do token | 401 | `Problema` | edge case | contrato |

**Não existe 403 nesta rota.** Não há como pedir ingresso de terceiro: o recorte é o
`sub`, aplicado na consulta. Ingresso alheio simplesmente não está no conjunto — não é
negado, é inalcançável (FR-014).

---

## 3. API — `POST /api/v1/ingressos/validar`

| Situação | Status | Corpo | Requisito | Teste |
|---|---|---|---|---|
| Ingresso válido | 200 | `valido: true` + `ingresso_id` + `utilizado_em` | FR-007 | contrato + integração |
| Ingresso já utilizado | 409 | `valido: false`, "já utilizado" | FR-008 | contrato |
| Ingresso cancelado | 409 | `valido: false`, "não está válido" | FR-009 | contrato |
| Código inexistente | 404 | `valido: false` | FR-010 | contrato |
| Código malformado | 404 | `valido: false` — **mesma resposta** | FR-010 | contrato |
| Assinatura inválida | 404 | `valido: false` — **mesma resposta** | FR-010 | contrato |
| `X-API-Key` ausente | 401 | `Problema` | FR-012 | contrato |
| `X-API-Key` não reconhecida | 401 | `Problema` | FR-012 | contrato |
| Apresenta token em vez da chave | 401 | `Problema` | edge case | contrato |
| Corpo sem `codigo_qr` | 422 | `Problema` | contrato de entrada | contrato |
| Duas leituras simultâneas do mesmo código | 200 + 409 | uma de cada | FR-011 | integração |

**Três exigências de indistinguibilidade, cada uma com teste próprio:**

1. Malformado, assinatura inválida e inexistente devolvem 404 com **corpo idêntico**.
   O teste compara as três respostas byte a byte — porque a diferença que vaza costuma
   ser uma palavra na mensagem, não o código de status.
2. Chave ausente e chave errada devolvem 401 idêntico.
3. A recusa por credencial (401) acontece **antes** de qualquer consulta ao ingresso
   (FR-012): sem chave válida, não se descobre nem que o código existe.

Cancelado devolve 409 e não 404 porque o ingresso **existe** e o portador tem direito
de saber que o bilhete dele foi cancelado, e não que nunca existiu. É informação sobre
o próprio ingresso apresentado, não sobre o acervo.

---

## 4. Consumo de `pagamento.sucesso`

Interface exposta, ainda que por evento — cada linha tem teste (princípio II).

| Situação | Desfecho | Requisito | Teste |
|---|---|---|---|
| Anúncio válido, reserva nova | emite, avisa, `Ack` | FR-003 | uso + integração |
| Anúncio válido, reserva já com ingresso | `Ack` inerte, sem avisar | FR-004 | uso + integração |
| Duas entregas simultâneas da mesma reserva | um único ingresso | FR-004 | integração |
| JSON ilegível | quarentena, sem retentativa | FR-002 | uso |
| `reserva_id`, `usuario_id`, `transacao_id` ou `pago_em` ausente | quarentena | FR-002 | uso |
| UUID malformado ou `pago_em` fora do RFC 3339 | quarentena | FR-002 | uso |
| Campos extras desconhecidos | processa normalmente | D1 | uso |
| Banco indisponível | nova tentativa | FR-022 | uso |
| Limite de entregas esgotado | fila morta | FR-022 | integração |
| **Canal de aviso com erro** | **`Ack`** + registro `FALHA` | FR-018, FR-025 | uso |

A última linha é a que mais importa proteger. Se o erro do notificador algum dia
escapar e virar `Nack`, o serviço passa a reprocessar mensagens por causa de falha de
e-mail — e a FR-018 morre sem que nenhum teste de sucesso perceba. O teste que a
sustenta usa um notificador falso que sempre falha e afirma **o desfecho da mensagem**,
não só o estado do banco.

---

## 5. Falhas de largada (o processo não sobe)

Não são respostas: são recusas de inicialização.

| Situação | Comportamento |
|---|---|
| Variável obrigatória ausente ou malformada | erro listando **todas** as chaves problemáticas de uma vez, e saída |
| `INGRESSO_QR_SEGREDO` ausente | idem — subir sem ele emitiria ingressos que a portaria nunca validaria (D11) |
| Topologia AMQP não declarável | erro e saída — consumir sem fila morta é perder mensagem em silêncio |
| Banco inalcançável na largada | erro e saída |
| JWKS inalcançável na largada | erro e saída |
