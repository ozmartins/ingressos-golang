GO ?= go
PKG := ./...

.PHONY: build test test-integration lint proto migrate-up migrate-down run certs openapi-sync \
        publicar-sessao publicar-pagamento contrato-sync

build:
	$(GO) build -o bin/estoque ./cmd/estoque

test:
	$(GO) test -race $(PKG)

test-integration:
	$(GO) test -race -tags=integration -timeout=20m ./test/integration/...

lint:
	golangci-lint run

# O .proto versionado em specs/ é a ORIGEM da cópia em proto/, que é o que o
# gerador consome; o alvo abaixo mantém as duas em sincronia. Origem não é fonte
# da verdade: quem descreve o comportamento do serviço é o código (constituição,
# princípio IX). Divergência entre os dois vira pergunta, não conserto unilateral
# (princípio X).
CONTRATO := specs/001-estoque-bloqueio-poltronas/contracts/estoque.proto

contrato-sync:
	cp $(CONTRATO) proto/estoque.proto

proto: contrato-sync
	buf generate

# O contrato versionado é a fonte da verdade; a cópia em internal/ existe só
# porque go:embed não alcança diretórios acima do pacote. O teste de paridade
# do pacote openapi falha quando as duas divergem.
CONTRATO := specs/001-estoque-bloqueio-poltronas/contracts/openapi.yaml

openapi-sync:
	cp $(CONTRATO) internal/adapter/http/openapi/openapi.yaml

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

certs:
	./scripts/gerar-certs.sh

run: build
	./bin/estoque

# Publicação manual de fatos, para o roteiro de quickstart.md.
SESSAO ?= f781a9b2-11e2-4f81-a901-8890bc123456
FILEIRAS ?= A,B
ASSENTOS ?= 10
RESULTADO ?= sucesso

publicar-sessao:
	$(GO) run ./cmd/publicar sessao-criada -sessao=$(SESSAO) -fileiras=$(FILEIRAS) -assentos=$(ASSENTOS)

publicar-pagamento:
	$(GO) run ./cmd/publicar pagamento -reserva=$(RESERVA) -resultado=$(RESULTADO)
