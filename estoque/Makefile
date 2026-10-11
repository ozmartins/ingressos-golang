GO ?= go
PKG := ./...

.PHONY: build test test-integration lint migrate-up migrate-down run \
        publicar-sessao publicar-pagamento

build:
	$(GO) build -o bin/estoque ./cmd/estoque

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
MIGRATE_URL ?= $(DATABASE_URL)&search_path=estoque

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

run: build
	./bin/estoque

# Publicação manual de fatos.
SESSAO ?= f781a9b2-11e2-4f81-a901-8890bc123456
FILEIRAS ?= A,B
ASSENTOS ?= 10
RESULTADO ?= sucesso

publicar-sessao:
	$(GO) run ./cmd/publicar sessao-criada -sessao=$(SESSAO) -fileiras=$(FILEIRAS) -assentos=$(ASSENTOS)

publicar-pagamento:
	$(GO) run ./cmd/publicar pagamento -reserva=$(RESERVA) -resultado=$(RESULTADO)
