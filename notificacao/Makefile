GO ?= go
PKG := ./...

.PHONY: build test test-integration lint migrate-up migrate-down run \
        publicar-pagamento publicar-cru

build:
	$(GO) build -o bin/notificacao ./cmd/notificacao

test:
	$(GO) test -race $(PKG)

test-integration:
	$(GO) test -race -tags=integration -timeout=20m ./test/integration/...

lint:
	golangci-lint run

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

run: build
	./bin/notificacao

# Publicação manual de pagamento.sucesso, para o roteiro de quickstart.md.
RESERVA ?=
USUARIO ?=
PAYLOAD ?=

publicar-pagamento:
	$(GO) run ./cmd/publicar -reserva=$(RESERVA) -usuario=$(USUARIO)

publicar-cru:
	$(GO) run ./cmd/publicar -cru='$(PAYLOAD)'
