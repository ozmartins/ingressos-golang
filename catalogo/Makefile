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

# A tabela de controle do migrate fica em `public`, fixada pelo `search_path`.
# O padrão (`"$$user", public`) a moveria para o schema `catalogo` assim que a
# primeira migração o criasse — o usuário do banco tem o mesmo nome —, e as
# versões já aplicadas sumiriam do rastro. As migrações em si qualificam cada
# objeto, então não dependem do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=public

migrate-up:
	migrate -path migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path migrations -database "$(MIGRATE_URL)" down 1

run: build
	./bin/catalogo
