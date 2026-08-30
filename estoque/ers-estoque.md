# Especificação do Microsserviço: Servico-Estoque

## 1. Visão Geral

O **`Servico-Estoque`** é o componente central para gestão de assentos e controle de concorrência em tempo real do sistema de cinema. Ele é responsável por manter a matriz de poltronas de cada sessão, processar solicitações de bloqueio temporário de alta velocidade via gRPC e reagir assincronamente a eventos de pagamento via RabbitMQ.

### Responsabilidades
* Gerenciar o estado de cada poltrona de uma sessão (`LIVRE`, `RESERVADA`, `OCUPADA`).
* Processar requisições síncronas de reserva/bloqueio via gRPC em milissegundos.
* Publicar o evento de reserva criada (`reserva.criada`) no RabbitMQ.
* Consumir eventos de falha no pagamento (`pagamento.falhou`) ou de confirmação (`pagamento.sucesso`) para atualizar o estado das poltronas.

---

## 2. Tecnologias e Dependências

* **Linguagem:** Go (Golang) 1.22+
* **Arquitetura:** Clean Architecture / Hexagonal Architecture
* **Protocolos:** gRPC (HTTP/2 / Protobuf) como servidor síncrono; AMQP (RabbitMQ) para eventos assíncronos.
* **Banco de Dados / Cache:** PostgreSQL (persistência) e Redis (controle de locks atômicos e expiração de reservas de 10 minutos).
* **Mensageria:** RabbitMQ

---

## 3. Modelo de Dados (Entidades do Banco de Dados)

### 3.1. `poltronas`
```sql
CREATE TABLE poltronas (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4 (ex: UUID único ou hash de sessao_id + fileira + numero)
    sessao_id VARCHAR(36) NOT NULL,
    fileira VARCHAR(5) NOT NULL, -- Ex: 'A', 'B', 'C'
    numero INT NOT NULL, -- Ex: 1, 2, 3
    tipo VARCHAR(50) NOT NULL DEFAULT 'NORMAL', -- NORMAL, PCD, NAMORADEIRA
    status VARCHAR(50) NOT NULL DEFAULT 'LIVRE', -- LIVRE, RESERVADA, OCUPADA
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_sessao_poltrona UNIQUE(sessao_id, fileira, numero)
);
```

### 3.2. `reservas`
```sql
CREATE TABLE reservas (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    sessao_id VARCHAR(36) NOT NULL,
    usuario_id VARCHAR(36) NOT NULL, -- UUID v4 originado do Keycloak
    expira_em TIMESTAMP WITH TIME ZONE NOT NULL, -- Agora + 10 minutos
    status VARCHAR(50) NOT NULL DEFAULT 'PENDENTE', -- PENDENTE, CONFIRMADA, EXPIRADA, CANCELADA
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3.3. `reserva_poltronas` (Tabela de Junção)
```sql
CREATE TABLE reserva_poltronas (
    reserva_id VARCHAR(36) NOT NULL REFERENCES reservas(id),
    poltrona_id VARCHAR(36) NOT NULL REFERENCES poltronas(id),
    PRIMARY KEY (reserva_id, poltrona_id)
);
```

---

## 4. Interface gRPC (Servidor)

O `Servico-Estoque` expõe o servidor gRPC para chamadas síncronas de altíssima performance provenientes do `Servico-Catalogo`.

### Arquivo Protobuf (`estoque.proto`)
```protobuf
syntax = "proto3";

package estoque;

option go_package = "./pb";

service ServicoEstoque {
  rpc BloquearPoltronas (SolicitacaoBloqueio) returns (RespostaBloqueio);
}

message SolicitacaoBloqueio {
  string sessao_id = 1;
  repeated string poltronas_ids = 2; // Lista de IDs das poltronas solicitadas
  string usuario_id = 3;             // UUID do usuário autenticado no Keycloak
}

message RespostaBloqueio {
  bool sucesso = 1;
  string reserva_id = 2;
  string mensagem = 3;
  int64 expira_em = 4;              // Timestamp Unix de expiração
}
```

---

## 5. Integração com RabbitMQ (Eventos)

### 5.1. Evento Publicado (Producer)
Ao realizar o bloqueio síncrono das poltronas com sucesso via gRPC, o serviço publica a seguinte mensagem no RabbitMQ:

* **Exchange:** `cinema.eventos` (Topic)
* **Routing Key:** `reserva.criada`
* **Payload JSON:**
```json
{
  "evento": "RESERVA_CRIADA",
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "sessao_id": "f781a9b2-11e2-4f81-a901-8890bc123456",
  "usuario_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
  "poltronas_ids": ["A1", "A2"],
  "expira_em": "2026-08-29T21:43:00Z"
}
```

### 5.2. Eventos Consumidos (Consumer)

#### A. Fila `estoque.pagamento-sucesso` (Binding: `pagamento.sucesso`)
* **Ação:** Atualiza o status da `reserva` para `CONFIRMADA` e o status das `poltronas` associadas para `OCUPADA` de forma definitiva.

#### B. Fila `estoque.pagamento-falhou` (Binding: `pagamento.falhou`)
* **Ação:** Atualiza o status da `reserva` para `CANCELADA`, libera o lock do Redis e altera o status das `poltronas` de volta para `LIVRE`.

---

## 6. Requisitos Não-Funcionais e Regras de Negócio

1. **Prevenção de Race Conditions:** O bloqueio de poltronas deve ser feito de forma atômica utilizando `Redis Distributed Lock` (ex: `SETNX`) ou transações com `SELECT FOR UPDATE` no PostgreSQL para evitar *double-booking*.
2. **Tempo de Resposta gRPC:** A resposta da RPC `BloquearPoltronas` deve ser inferior a **100 milissegundos**.
3. **Gerenciamento de Timeouts (TTL):** Reservas não pagas dentro de 10 minutos devem ser invalidadas automaticamente, liberando as poltronas vinculadas.
4. **Idempotência:** O processamento das mensagens de eventos consumidos do RabbitMQ deve garantir idempotência usando a chave `reserva_id`.
