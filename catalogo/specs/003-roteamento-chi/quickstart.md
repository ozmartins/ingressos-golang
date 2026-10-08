# Quickstart: validar a troca para chi

Pré-requisitos: Go 1.25+ e, para o passo 3, Docker. Rodar de dentro de `ingressos-golang/catalogo/`.

## 1. Núcleo isolado e dependências (sem banco)

```
make test     # go test -race ./...  (inclui paridade com o OpenAPI e os testes novos de roteamento)
make lint     # depguard agora proíbe github.com/go-chi no domínio e nos casos de uso (SC-004)
```

## 2. Contrato HTTP (SC-001, SC-002)

```
go test -race ./test/contract/...
```

Esperado: as mesmas suítes de antes, asserções inalteradas.

## 3. Borda de roteamento (US2) com o serviço de pé

```
docker compose up --build catalogo
curl -i -X POST  localhost:<porta>/api/v1/filmes/x   # 405, Allow completo, corpo "Method Not Allowed"
curl -I          localhost:<porta>/api/v1/filmes     # 200 (HEAD atendido pela rota GET)
curl -i          localhost:<porta>/nada              # 404 page not found
curl -i          localhost:<porta>/docs/             # 200
```

(`<porta>` em `URLS.txt`.)

## 4. Rótulo de métrica/log (achado D3)

Chamar `GET /api/v1/filmes/<uuid>` e conferir no log "requisição atendida" que
`rota="GET /api/v1/filmes/{id}"` — nunca o caminho com o identificador.
