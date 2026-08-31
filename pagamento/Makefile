GO ?= go
PKG := ./...

.PHONY: build test test-integration lint migrate-up migrate-down run \
        publicar-reserva publicar-rajada espiar-evento

build:
	$(GO) build -o bin/pagamento ./cmd/pagamento

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
	./bin/pagamento

# Publicação manual de reserva.criada, para o roteiro de quickstart.md.
# Necessário porque o Servico-Estoque ainda não inclui valor_total nem
# forma_pagamento no fato (research.md D1).
RESERVA ?=
VALOR ?= 84.00
FORMA ?= PIX
EXPIRA_EM ?= 10m
N ?= 1000

publicar-reserva:
	$(GO) run ./cmd/publicar -reserva=$(RESERVA) -valor=$(VALOR) -forma=$(FORMA) -expira-em=$(EXPIRA_EM)

publicar-rajada:
	$(GO) run ./cmd/publicar -rajada=$(N) -valor=$(VALOR) -forma=$(FORMA) -expira-em=$(EXPIRA_EM)

ROUTING ?= pagamento.sucesso
espiar-evento:
	$(GO) run ./cmd/publicar -espiar=$(ROUTING)
