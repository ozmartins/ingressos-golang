# Quickstart — Validação do Servico-Catalogo

**Feature**: `001-catalogo-sessoes-reserva` | **Date**: 2026-08-29

Como subir o serviço e provar, na prática, que a feature funciona ponta a ponta.
Os detalhes de contrato estão em [`contracts/openapi.yaml`](./contracts/openapi.yaml)
e os de dados em [`data-model.md`](./data-model.md) — este guia não os repete.

## Pré-requisitos

- Go 1.22+
- Docker (Postgres, Keycloak e o estoque simulado sobem em contêiner)
- `protoc` com `protoc-gen-go` e `protoc-gen-go-grpc`, para regerar `gen/pb`
- `migrate` (golang-migrate) para aplicar o esquema

## Subir as dependências

```bash
docker compose up -d          # postgres, keycloak, estoque simulado
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/catalogo
```

O processo **falha ao subir** se faltar variável obrigatória — comportamento esperado (FR-032).
Variáveis mínimas: `DATABASE_URL`, `KEYCLOAK_ISSUER_URL`, `KEYCLOAK_AUDIENCE`, `ESTOQUE_GRPC_ADDR`.
A lista completa, com padrões, está em `research.md` (D10).

Carregar o catálogo de exemplo:

```bash
psql "$DATABASE_URL" -f test/fixtures/catalogo_exemplo.sql
```

## Roteiro de validação

Cada cenário mapeia para uma história da spec. Rode-os na ordem — o último depende dos anteriores para ter um `sessao_id` real.

### 1. Descobrir filmes (User Story 1)

```bash
curl -s "localhost:8080/api/v1/filmes" | jq
curl -s "localhost:8080/api/v1/filmes?status=EM_CARTAZ&page=1&page_size=5" | jq
curl -si "localhost:8080/api/v1/filmes?status=INEXISTENTE" | head -1
curl -si "localhost:8080/api/v1/filmes?page_size=500" | head -1
```

**Esperado**: sem filtro, só `EM_CARTAZ` e `BREVE`; a resposta é um objeto com `itens` e `pagina`, nunca um array nu; `status` desconhecido e `page_size` acima do teto retornam `400` em `application/problem+json`.

### 2. Grade de sessões e paginação (User Story 2)

```bash
curl -s "localhost:8080/api/v1/sessoes?page_size=2" | jq '.pagina'
curl -s "localhost:8080/api/v1/sessoes?page=2&page_size=2" | jq '.itens[].id'
curl -s "localhost:8080/api/v1/sessoes?data=2026-09-01&filme_id=<uuid>" | jq
curl -si "localhost:8080/api/v1/sessoes?data=01-09-2026" | head -1
```

**Esperado**: `pagina.total` reflete todos os registros filtrados, não os da página; os `id` da página 2 não repetem os da página 1 (FR-004); sessões `CANCELADA` e `FINALIZADA` nunca aparecem; data malformada retorna `400`.

Para confirmar a página vazia além do fim (FR-005): peça `page=9999` e verifique `itens: []` com `total` correto e status `200`.

### 3. Cinemas e salas (User Story 4)

```bash
curl -s "localhost:8080/api/v1/cinemas" | jq '.itens[0]'
curl -s "localhost:8080/api/v1/cinemas/<uuid>/salas" | jq
curl -si "localhost:8080/api/v1/cinemas/00000000-0000-0000-0000-000000000000/salas" | head -1
```

**Esperado**: cinema inexistente retorna `404` com `type: .../cinema-nao-encontrado`.

### 4. Reserva de poltronas (User Story 3)

Obter um token do Keycloak local:

```bash
TOKEN=$(curl -s -d "client_id=cinema-app" -d "username=teste" -d "password=teste" \
  -d "grant_type=password" \
  "$KEYCLOAK_ISSUER_URL/protocol/openid-connect/token" | jq -r .access_token)
```

Caminho feliz e as recusas:

```bash
# 201 — reserva confirmada
curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["A1","A2"]}' | head -1

# 401 — sem credencial; o estoque NÃO deve ser contatado
curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H 'Content-Type: application/json' -d '{"poltronas_ids":["A1"]}' | head -1

# 400 — lista vazia e lista com duplicatas
curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["A1","A1"]}' | head -1

# 409 — poltrona já ocupada (o estoque simulado marca B1 como ocupada)
curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["B1"]}' | head -1

# 404 — sessão inexistente; o estoque NÃO deve ser contatado
curl -si -X POST "localhost:8080/api/v1/sessoes/00000000-0000-0000-0000-000000000000/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["A1"]}' | head -1
```

**Como confirmar que o estoque não foi contatado** nos casos 401 e 404: o contador `estoque.bloqueio.total` em `/metrics` não deve variar entre antes e depois da chamada. É isso que prova os cenários 3 e 5 da User Story 3 — o status HTTP sozinho não distingue "recusou antes" de "recusou depois".

### 5. Falha do estoque: timeout e recusa rápida (SC-004, SC-007)

```bash
docker compose stop estoque-simulado

# 1ª a 5ª chamadas: ~2s cada (timeout), status 503
time curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["A3"]}' | head -1

# 6ª em diante: < 200 ms (recusa rápida), mesmo 503 e mesmo corpo
time curl -si -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"poltronas_ids":["A4"]}' | head -1

# navegação segue funcionando com o estoque fora do ar (SC-012)
curl -s "localhost:8080/api/v1/sessoes" | jq '.pagina.total'

docker compose start estoque-simulado
# após o intervalo configurado (padrão 30s), a reserva volta a funcionar sozinha
```

**Esperado**: o corpo do 503 é idêntico nos dois casos — o cliente não distingue timeout de recusa rápida (exigência da spec); só as métricas distinguem.

### 6. Correlação ponta a ponta (SC-011)

```bash
curl -s -X POST "localhost:8080/api/v1/sessoes/<sessao_id>/reservar" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' \
  -d '{"poltronas_ids":["A5"]}'
```

**Esperado**: todos os logs da requisição carregam `trace_id=4bf92f3577b34da6a3ce929d0e0e4736`, e o mesmo `trace_id` aparece do lado do estoque simulado — provando que o contexto foi propagado (FR-036).

## Suíte automatizada

```bash
go test ./...                                  # unitários — domínio e casos de uso, sem infra
go test -tags=integration ./test/integration/  # Postgres real + estoque em bufconn
go test -race ./...                            # concorrência (SC-006)
```

O teste de concorrência de SC-006 dispara 50 solicitações paralelas sobre as mesmas poltronas contra um estoque simulado que só concede o primeiro pedido, e verifica que exatamente uma recebe `201` e as demais `409`.

## Saúde e sinais

```bash
curl -s localhost:8080/health | jq
curl -s localhost:8080/metrics | grep estoque_
```

`/health` retorna `503` quando o banco está inacessível — o orquestrador tira a instância de rotação (FR-037). O estoque fora do ar **não** derruba a saúde: a navegação continua servível (SC-012).
