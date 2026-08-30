# Especificação do Microsserviço: Servico-Pagamento

## 1. Visão Geral

O **`Servico-Pagamento`** é o microsserviço responsável pelo processamento assíncrono das transações financeiras do sistema de cinema. Ele atua amortecendo picos de acesso (*load leveling*), consumindo intenções de compra publicadas no RabbitMQ e realizando a integração com adquirentes/gateways de pagamento sem bloquear a navegação do usuário.

### Responsabilidades
* Consumir eventos de reserva criada (`reserva.criada`) no RabbitMQ.
* Processar a cobrança (PIX ou Cartão de Crédito) de forma desacoplada.
* Publicar eventos de resultado do pagamento (`pagamento.sucesso` ou `pagamento.falhou`) para os demais serviços.
* Expor uma API REST para consulta do status do pagamento pelo aplicativo/front-end.

---

## 2. Tecnologias e Dependências

* **Linguagem:** Go (Golang) 1.22+
* **Arquitetura:** Clean Architecture / Hexagonal Architecture
* **Protocolos:** AMQP (RabbitMQ) para eventos assíncronos; REST (HTTP/JSON) para consultas de status.
* **Autenticação:** Validação stateless de JWT (emitido pelo Keycloak) no endpoint REST.
* **Banco de Dados:** PostgreSQL (armazenamento do histórico e status das transações de pagamento).
* **Mensageria:** RabbitMQ

---

## 3. Modelo de Dados (Entidades do Banco de Dados)

### 3.1. `transacoes_pagamento`
```sql
CREATE TABLE transacoes_pagamento (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    reserva_id VARCHAR(36) NOT NULL UNIQUE, -- UUID v4 vindo da reserva
    usuario_id VARCHAR(36) NOT NULL, -- UUID v4 do Keycloak
    valor_total DECIMAL(10, 2) NOT NULL,
    forma_pagamento VARCHAR(50) NOT NULL, -- PIX, CARTAO_CREDITO
    status VARCHAR(50) NOT NULL DEFAULT 'PROCESSANDO', -- PROCESSANDO, PAGO, RECUSADO, CANCELADO
    codigo_transacao_gateway VARCHAR(255), -- ID de referência retornado pelo gateway externo
    motivo_falha TEXT,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Integração com RabbitMQ (Eventos)

### 4.1. Evento Consumido (Consumer)

#### Fila `pagamento.reserva-criada` (Binding: `reserva.criada`)
* **Ação:** Inicia o fluxo de cobrança. Registra a transação como `PROCESSANDO` e simula/executa a chamada ao gateway de pagamento.
* **Payload JSON Esperado:**
```json
{
  "evento": "RESERVA_CRIADA",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "poltronas_ids": ["A1", "A2"],
  "valor_total": 84.00,
  "forma_pagamento": "PIX",
  "expira_em": "2026-08-29T21:43:00Z"
}
```

---

### 4.2. Eventos Publicados (Producer)

#### A. Evento de Sucesso
* **Exchange:** `cinema.eventos` (Topic)
* **Routing Key:** `pagamento.sucesso`
* **Payload JSON:**
```json
{
  "evento": "PAGAMENTO_SUCESSO",
  "transacao_id": "e402a129-8812-4211-b123-000129381293",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "valor_total": 84.00,
  "pago_em": "2026-08-29T21:35:10Z"
}
```

#### B. Evento de Falha
* **Exchange:** `cinema.eventos` (Topic)
* **Routing Key:** `pagamento.falhou`
* **Payload JSON:**
```json
{
  "evento": "PAGAMENTO_FALHOU",
  "transacao_id": "e402a129-8812-4211-b123-000129381293",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "motivo": "SALDO_INSUFICIENTE"
}
```

---

## 5. Contratos de API (REST - HTTP)

### Endpoints Protegidos (Requer Token JWT no Header `Authorization: Bearer <token>`)

#### `GET /api/v1/pagamentos/reserva/{reserva_id}`
Permite ao aplicativo/front-end consultar o status do processamento do pagamento.

* **Headers:** `Authorization: Bearer <JWT_KEYCLOAK>`
* **Path Parameter:** `reserva_id` (UUID da reserva)
* **Response `200 OK`:**
```json
{
  "transacao_id": "e402a129-8812-4211-b123-000129381293",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "status": "PAGO",
  "valor_total": 84.00,
  "forma_pagamento": "PIX",
  "criado_em": "2026-08-29T21:35:00Z"
}
```

---

## 6. Requisitos Não-Funcionais e Regras de Negócio

1. **Garantia de Idempotência:** O consumidor do RabbitMQ deve verificar se a `reserva_id` já foi processada antes de executar qualquer cobrança, evitando cobranças duplicadas.
2. **Nivelamento de Carga (*Load Leveling*):** O consumidor deve utilizar o atributo `Prefetch Count` no RabbitMQ (ex: máximo de 10 mensagens concorrentes) para respeitar o limite de taxa do gateway de pagamento.
3. **Tratamento de Mensagens Mortas (Dead Letter Exchange - DLX):** Caso o processamento do pagamento lance uma exceção não tratada ou erro de infraestrutura, a mensagem deve ser enviada para uma fila de *Dead Letter* após 3 tentativas incompletas.
4. **Resiliência:** Em caso de falha de conexão com o banco de dados PostgreSQL durante o consumo da fila, a mensagem deve ser rejeitada e reenfileirada (`nack` com requeue).
