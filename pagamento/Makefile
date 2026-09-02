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

# A tabela de controle do migrate (`schema_migrations`) fica no schema do
# serviço, fixada pelo `search_path`, e não em `public`: o banco `cinema` é
# compartilhado pelos quatro serviços, e em `public` os quatro disputariam a
# mesma tabela. As migrações em si qualificam cada objeto, então não dependem
# do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=pagamento

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
