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

# A tabela de controle do migrate fica em `public`, fixada pelo `search_path`.
# O padrão (`"$$user", public`) a moveria para o schema `notificacao` assim que
# a primeira migração o criasse, caso o usuário do banco tenha o mesmo nome, e
# as versões já aplicadas sumiriam do rastro. As migrações em si qualificam cada
# objeto, então não dependem do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=public

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

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
