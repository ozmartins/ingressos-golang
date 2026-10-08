# Quickstart: validar o roteamento via chi

Pré-requisito: estar em `ingressos-golang/pagamento`.

1. **Testes**: `make test` — toda a suíte passa sem asserções enfraquecidas (SC-001), incluindo os testes novos de 404/405, `/docs/`, `HEAD` e `reserva_id`.
2. **Lint e fronteira do núcleo**: `make lint` — passa, com `go-chi/chi` na regra `nucleo-sem-adaptadores` (SC-004). Conferir: `grep -r go-chi internal/domain internal/usecase` não retorna nada.
3. **Contrato**: os testes de `openapi_test.go` passam sem alterar `specs/001-pagamento-assincrono/contracts/openapi.yaml` (FR-007).
4. **Fluxo ponta a ponta** (opcional, `docker compose up --build` na raiz): `GET /api/v1/health/live` → 200; `GET /docs` e `/docs/` → UI; `GET /api/v1/pagamentos/reserva/<uuid>` sem token → 401; com token de dona → 200; `POST` com forma válida → 202.
5. **Comparação 404/405**: `curl -i` em caminho inexistente e em `DELETE` numa rota existente; status, `Allow` e corpo iguais aos registrados nos testes de caracterização.
