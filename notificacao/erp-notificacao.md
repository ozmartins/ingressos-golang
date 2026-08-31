# Especificação do Microsserviço: Servico-Notificacao

## 1. Visão Geral

O **`Servico-Notificacao`** é o microsserviço responsável pela emissão dos ingressos digitais e envio das confirmações de compra para os usuários. Ele atua de forma totalmente desacoplada, reagindo a eventos de pagamentos confirmados via RabbitMQ para gerar o QR Code de acesso e disponibilizar os bilhetes para a portaria do cinema e para o aplicativo do cliente.

### Responsabilidades
* Consumir eventos de pagamento bem-sucedido (`pagamento.sucesso`) no RabbitMQ.
* Gerar os ingressos digitais com hash único e payload do QR Code para entrada no cinema.
* Registrar o log de envio de confirmações aos clientes.
* Expor API REST para consulta dos ingressos do cliente e para validação do QR Code na catraca/portaria do cinema.

---

## 2. Tecnologias e Dependências

* **Linguagem:** Go (Golang) 1.22+
* **Arquitetura:** Clean Architecture / Hexagonal Architecture
* **Protocolos:** AMQP (RabbitMQ) para eventos assíncronos; REST (HTTP/JSON) para leitura de ingressos e validação.
* **Autenticação:** Validação stateless de JWT (emitido pelo Keycloak) para o cliente e chave de API / escopo específico para a portaria do cinema.
* **Banco de Dados:** PostgreSQL (armazenamento dos ingressos emitidos e registros de validação).
* **Mensageria:** RabbitMQ

---

## 3. Modelo de Dados (Entidades do Banco de Dados)

### 3.1. `ingressos_emitidos`
```sql
CREATE TABLE ingressos_emitidos (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    reserva_id VARCHAR(36) NOT NULL UNIQUE, -- UUID v4 vindo do evento de pagamento
    usuario_id VARCHAR(36) NOT NULL, -- UUID v4 do Keycloak
    codigo_qr VARCHAR(255) NOT NULL UNIQUE, -- Hash/Payload assinado contido no QR Code
    status VARCHAR(50) NOT NULL DEFAULT 'VALIDO', -- VALIDO, UTILIZADO, CANCELADO
    utilizado_em TIMESTAMP WITH TIME ZONE,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2. `registros_notificacao`
```sql
CREATE TABLE registros_notificacao (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    ingresso_id VARCHAR(36) NOT NULL REFERENCES ingressos_emitidos(id),
    usuario_id VARCHAR(36) NOT NULL,
    canal VARCHAR(50) NOT NULL DEFAULT 'EMAIL', -- EMAIL, PUSH, SMS
    status VARCHAR(50) NOT NULL DEFAULT 'ENVIADO', -- ENVIADO, FALHA
    detalhes TEXT,
    enviado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Integração com RabbitMQ (Eventos)

### Evento Consumido (Consumer)

#### Fila `notificacao.pagamento-sucesso` (Binding: `pagamento.sucesso`)
* **Ação:** Cria o registro do `ingresso_emitido`, gera o token único do `codigo_qr` e registra o disparo fictício/real da notificação.
* **Payload JSON Esperado:**
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

---

## 5. Contratos de API (REST - HTTP)

### 5.1. Endpoints do Cliente (Requer Token JWT no Header `Authorization: Bearer <token>`)

#### `GET /api/v1/ingressos/meus-ingressos`
Lista todos os ingressos válidos e históricos do usuário autenticado.

* **Headers:** `Authorization: Bearer <JWT_KEYCLOAK>`
* **Response `200 OK`:**
```json
[
  {
    "ingresso_id": "771a9210-9981-42a1-b882-102938471290",
    "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
    "codigo_qr": "CINEMA-TOKEN-SIGNED-9982a1b3-44c1",
    "status": "VALIDO",
    "criado_em": "2026-08-29T21:35:12Z"
  }
]
```

---

### 5.2. Endpoints Operacionais (Portaria/Validacao do Cinema)

#### `POST /api/v1/ingressos/validar`
Realiza a leitura e validação do QR Code na entrada da sala de cinema (baixa do ingresso).

* **Headers:** `X-API-Key: <CHAVE_DISPOSITIVO_PORTARIA>`
* **Request Body:**
```json
{
  "codigo_qr": "CINEMA-TOKEN-SIGNED-9982a1b3-44c1"
}
```
* **Response `200 OK` (Ingresso Válido):**
```json
{
  "valido": true,
  "mensagem": "Entrada autorizada.",
  "ingresso_id": "771a9210-9981-42a1-b882-102938471290",
  "utilizado_em": "2026-08-29T22:00:00Z"
}
```
* **Response `409 Conflict` (Ingresso Já Utilizado):**
```json
{
  "valido": false,
  "mensagem": "Ingresso já utilizado anteriormente."
}
```

---

## 6. Requisitos Não-Funcionais e Regras de Negócio

1. **Garantia de Emissão Única (Idempotência):** O consumidor do RabbitMQ deve usar o `reserva_id` como chave única de idempotência no banco de dados para evitar a emissão de ingressos duplicados para o mesmo pagamento.
2. **Segurança do QR Code:** O `codigo_qr` gerado não deve conter dados sensíveis em texto puro, podendo ser um hash assinado digitalmente (HMAC) para prevenir fraudes e falsificações na portaria.
3. **Resiliência na Notificação:** Falhas no envio de e-mail/notificação externa não devem reverter ou falhar a emissão do ingresso no banco de dados; o status da notificação deve apenas ser marcado como `FALHA` na tabela `registros_notificacao` para tentativa posterior.
