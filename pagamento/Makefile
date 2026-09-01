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

# A tabela de controle do migrate fica em `public`, fixada pelo `search_path`.
# O padrão (`"$$user", public`) a moveria para o schema `pagamento` assim que a
# primeira migração o criasse — o usuário do banco tem o mesmo nome —, e as
# versões já aplicadas sumiriam do rastro. A migração em si qualifica cada
# objeto, então não depende do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=public

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

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
