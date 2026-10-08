# Quickstart: validar o roteamento chi

## Pré-requisitos

- Go 1.25; dentro de `ingressos-golang/notificacao`.

## Verificação automatizada

```
go mod tidy && go build ./...
go test -race ./internal/adapter/http/...
make test
make lint
```

Esperado: suíte existente passa sem alteração de asserções; `roteamento_test.go` passa.

## Verificação manual (opcional, com a plataforma no ar)

```
cd .. && docker compose up --build notificacao
curl -i localhost:<porta>/health/live          # 200 {"status":"vivo"}
curl -I localhost:<porta>/health/live          # 200 (HEAD)
curl -i localhost:<porta>/inexistente          # 404
curl -i localhost:<porta>/api/v1/ingressos/validar   # 405 com Allow: POST
curl -i localhost:<porta>/docs/                # 200 HTML
curl -i localhost:<porta>/api/v1/ingressos/meus-ingressos   # 401 problema+json
```

Portas em `../URLS.txt`.
