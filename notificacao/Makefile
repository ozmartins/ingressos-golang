GO ?= go
PKG := ./...

.PHONY: build test test-integration lint migrate-up migrate-down run openapi-sync \
        publicar-pagamento publicar-cru

build:
	$(GO) build -o bin/notificacao ./cmd/notificacao

test:
	$(GO) test -race $(PKG)

test-integration:
	$(GO) test -race -tags=integration -timeout=20m ./test/integration/...

lint:
	golangci-lint run

# O contrato versionado é a fonte da verdade; a cópia em internal/ existe só
# porque go:embed não alcança diretórios acima do pacote. O teste de paridade
# do pacote openapi falha quando as duas divergem.
CONTRATO := specs/001-emissao-ingressos/contracts/openapi.yaml

openapi-sync:
	cp $(CONTRATO) internal/adapter/http/openapi/openapi.yaml

# A tabela de controle do migrate (`schema_migrations`) fica no schema do
# serviço, fixada pelo `search_path`, e não em `public`: o banco `cinema` é
# compartilhado pelos quatro serviços, e em `public` os quatro disputariam a
# mesma tabela. As migrações em si qualificam cada objeto, então não dependem
# do `search_path`.
MIGRATE_URL ?= $(DATABASE_URL)&search_path=notificacao

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
