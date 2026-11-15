GO ?= go
PKG := ./...

# Regenera gen/pb a partir de proto/ (precisa de buf, protoc-gen-go e protoc-gen-go-grpc no PATH;
# costumam estar em ~/go/bin). O código gerado é commitado: o Dockerfile não roda buf.
.PHONY: proto build test test-integration lint migrate-up migrate-down run

proto:
	buf generate

build:
	$(GO) build -o bin/catalogo ./cmd/catalogo

test:
	$(GO) test -race $(PKG)

test-integration:
	$(GO) test -race -tags=integration ./test/integration/...

lint:
	golangci-lint run

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
