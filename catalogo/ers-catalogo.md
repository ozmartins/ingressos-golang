# Especificação do Microsserviço: Servico-Catalogo

## 1. Visão Geral

O **`Servico-Catalogo`** é o ponto de entrada para os clientes navegarem pelo catálogo de filmes, complexos de cinema, salas e sessões disponíveis. Ele é responsável por expor uma API REST pública para o aplicativo/front-end e atuar como cliente gRPC síncrono para o `Servico-Estoque` quando um usuário inicia a seleção/reserva de ingressos.

### Responsabilidades
* Gerenciar e expor a listagem de filmes em cartaz e em breve.
* Expor informações sobre cinemas, salas e suas configurações.
* Consultar e expor a grade de sessões por filme, cinema ou data.
* Intermediar a intenção de compra do usuário fazendo a chamada gRPC síncrona de reserva ao `Servico-Estoque`.

---

## 2. Tecnologias e Dependências

* **Linguagem:** Go (Golang) 1.22+
* **Arquitetura:** Clean Architecture / Hexagonal Architecture
* **Protocolos:** REST (HTTP/JSON) para clientes externos; gRPC (HTTP/2 / Protobuf) como cliente interno.
* **Autenticação:** Validação stateless de JWT (emitido pelo Keycloak) via cabeçalho `Authorization: Bearer <token>`.
* **Banco de Dados:** PostgreSQL (armazenamento das informações estruturais de filmes, cinemas, salas e sessões).

---

## 3. Modelo de Dados (Entidades do Banco de Dados)

### 3.1. `filmes`
```sql
CREATE TABLE filmes (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    titulo VARCHAR(255) NOT NULL,
    sinopse TEXT,
    duracao_minutos INT NOT NULL,
    classificacao_etaria VARCHAR(50) NOT NULL,
    genero VARCHAR(100) NOT NULL,
    imagem_url VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'EM_CARTAZ', -- EM_CARTAZ, BREVE, FORA_DE_CARTAZ
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    atualizado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2. `cinemas`
```sql
CREATE TABLE cinemas (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    nome VARCHAR(255) NOT NULL,
    cidade VARCHAR(100) NOT NULL,
    estado VARCHAR(2) NOT NULL,
    endereco TEXT NOT NULL,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3.3. `salas`
```sql
CREATE TABLE salas (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    cinema_id VARCHAR(36) NOT NULL REFERENCES cinemas(id),
    numero INT NOT NULL,
    tipo_tela VARCHAR(50) NOT NULL, -- 2D, 3D, IMAX, VIP
    capacidade_total INT NOT NULL,
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3.4. `sessoes`
```sql
CREATE TABLE sessoes (
    id VARCHAR(36) PRIMARY KEY, -- UUID v4
    filme_id VARCHAR(36) NOT NULL REFERENCES filmes(id),
    sala_id VARCHAR(36) NOT NULL REFERENCES salas(id),
    data_hora_inicio TIMESTAMP WITH TIME ZONE NOT NULL,
    idioma VARCHAR(50) NOT NULL, -- DUBLADO, LEGENDADO
    preco_base DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'AGENDADA', -- AGENDADA, EM_ANDAMENTO, FINALIZADA, CANCELADA
    criado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

---

## 4. Contratos de API (REST - HTTP)

### 4.1. Endpoints Públicos (Sem necessidade de autenticação)

#### `GET /api/v1/filmes`
Lista os filmes cadastrados.
* **Query Params:** `status` (opcional: `EM_CARTAZ`, `BREVE`)
* **Response `200 OK`:**
```json
[
  {
    "id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
    "titulo": "Duna: Parte 2",
    "sinopse": "Paul Atreides se une a Chani e aos Fremen...",
    "duracao_minutos": 166,
    "classificacao_etaria": "14 anos",
    "genero": "Ficção Científica",
    "imagem_url": "https://cdn.cinema.com/posters/duna2.jpg",
    "status": "EM_CARTAZ"
  }
]
```

#### `GET /api/v1/sessoes`
Busca a grade de sessões com filtros.
* **Query Params:** `filme_id` (opcional), `cinema_id` (opcional), `data` (opcional, formato `YYYY-MM-DD`)
* **Response `200 OK`:**
```json
[
  {
    "id": "f781a9b2-11e2-4f81-a901-8890bc123456",
    "filme_id": "c394c8b3-76a1-4328-b803-02f5923b7a15",
    "cinema_nome": "CineMark - Shopping Centro",
    "sala_numero": 3,
    "tipo_tela": "IMAX",
    "data_hora_inicio": "2026-09-01T20:30:00Z",
    "idioma": "LEGENDADO",
    "preco_base": 42.00
  }
]
```

---

### 4.2. Endpoints Protegidos (Requer Token JWT no Header `Authorization: Bearer <token>`)

#### `POST /api/v1/sessoes/{id}/reservar`
Solicita o bloqueio de poltronas para uma sessão específica. Este endpoint faz a ponte síncrona via gRPC com o `Servico-Estoque`.

* **Headers:** `Authorization: Bearer <JWT_KEYCLOAK>`
* **Path Parameter:** `id` (ID da sessão)
* **Request Body:**
```json
{
  "poltronas_ids": ["A1", "A2"]
}
```
* **Processamento Interno:**
  1. Extrai o `usuario_id` do campo `sub` do token JWT.
  2. Executa a chamada gRPC `BloquearPoltronas` para o `Servico-Estoque`.
* **Response `201 Created`:**
```json
{
  "sucesso": true,
  "reserva_id": "9982a1b3-44c1-4221-a123-902183120192",
  "mensagem": "Poltronas reservadas com sucesso.",
  "expira_em": 1788290000
}
```
* **Response `409 Conflict` (Poltrona já ocupada/reservada):**
```json
{
  "sucesso": false,
  "mensagem": "Uma ou mais poltronas selecionadas não estão disponíveis."
}
```

---

## 5. Interface gRPC (Cliente)

O `Servico-Catalogo` atuará como cliente do serviço de estoque utilizando o seguinte contrato de comunicação gRPC (`estoque.proto`):

```protobuf
syntax = "proto3";

package estoque;

option go_package = "./pb";

service ServicoEstoque {
  rpc BloquearPoltronas (SolicitacaoBloqueio) returns (RespostaBloqueio);
}

message SolicitacaoBloqueio {
  string sessao_id = 1;
  repeated string poltronas_ids = 2;
  string usuario_id = 3; // UUID extraído do token JWT pelo Servico-Catalogo
}

message RespostaBloqueio {
  bool sucesso = 1;
  string reserva_id = 2;
  string mensagem = 3;
  int64 expira_em = 4;
}
```

---

## 6. Requisitos Não-Funcionais e Regras de Negócio

1. **Validação de JWT:** As rotas protegidas devem validar a assinatura do token JWT emitido pelo Keycloak localmente usando a chave pública (algoritmo RS256/JWKS).
2. **Resiliência e Timeout no gRPC:** As chamadas gRPC de reserva para o `Servico-Estoque` devem possuir um *timeout* configurado para no máximo **2 segundos** para garantir resposta rápida ao usuário.
3. **Mapeamento de Erros:** Caso o `Servico-Estoque` esteja indisponível ou estoure o timeout, a API REST deve retornar HTTP status `503 Service Unavailable` com mensagem padronizada em JSON.
4. **Variáveis de Ambiente:** Todas as configurações de conexões (Banco de dados, URL do Keycloak, Endereço gRPC do `Servico-Estoque`) devem ser injetadas via variáveis de ambiente.
