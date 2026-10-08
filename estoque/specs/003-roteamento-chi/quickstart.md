# Quickstart: validar o roteamento via chi

Pré-requisito: `cd ingressos-golang/estoque`.

1. **Antes de trocar** — escrever o teste de caracterização (tabela da D8 em
   `research.md`) e rodá-lo contra o `ServeMux` atual: deve passar.
   `go test -race ./internal/adapter/http/... ./internal/platform/health/...`
2. **Depois de trocar** — a mesma tabela, sem alterar asserções, deve passar com o chi,
   exceto as linhas da D7 (307), documentadas como diferença conhecida.
3. **Suíte completa**: `make test` (inclui `test/arquitetura_test.go`, que passa a
   proibir `github.com/go-chi` no núcleo) e `make lint` (depguard).
4. **Integração**: `make test-integration` — inclui `largada_test.go`, que consulta
   `http://127.0.0.1:18090/health/ready`.
5. **Ponta a ponta** (opcional): `cd .. && docker compose up --build estoque`, depois
   `curl -i localhost:<porta>/docs`, `curl -I localhost:<porta>/docs` (HEAD → 200),
   `curl -i -X DELETE localhost:<porta>/docs` (405 + `Allow`) e
   `curl -i localhost:<porta>/health/ready` (porta de administração).
6. **Núcleo limpo**: `go list -deps ./internal/domain/... ./internal/usecase/... | grep go-chi`
   não pode listar nada.
