GO ?= go
PKG := ./...

.PHONY: build test test-integration lint proto openapi-sync migrate-up migrate-down run

build:
	$(GO) build -o bin/catalogo ./cmd/catalogo

test:
	$(GO) test -race $(PKG)

test-integration:
	$(GO) test -race -tags=integration ./test/integration/...

lint:
	golangci-lint run

proto:
	buf generate

# O contrato versionado em specs/ é a fonte da verdade; go:embed não alcança
# diretórios acima do pacote, então a cópia servida em /openapi.yaml vive junto
# do código. O teste de sincronia falha quando as duas divergem.
CONTRATO := specs/001-catalogo-sessoes-reserva/contracts/openapi.yaml

openapi-sync:
	cp $(CONTRATO) internal/adapter/http/openapi/openapi.yaml

# A tabela de controle do migrate (`schema_migrations`) fica no schema do
# serviço, fixada pelo `search_path`, e não em `public`: o banco `cinema` é
# compartilhado pelos quatro serviços, e em `public` os quatro disputariam a
# mesma tabela. As migrações em si qualificam cada objeto, então não dependem
# do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=catalogo

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

run: build
	./bin/catalogo
