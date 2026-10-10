# Quickstart: validar a troca para GORM

Pré-requisitos: Docker (Testcontainers e `docker compose`), Go 1.25+. Rodar de dentro de
`ingressos-golang/catalogo/`.

## 1. Núcleo isolado e dependências (sem banco)

```
make test     # go test -race ./...
make lint     # golangci-lint: depguard agora proíbe gorm.io no domínio e nos casos de uso (SC-005)
```

## 2. Equivalência com Postgres real (SC-001, SC-002)

```
make test-integration
```

Esperado: as mesmas suítes de antes, asserções inalteradas — `catalogo_test.go`,
`escrita_salas_sessoes_test.go`, `sessoes_test.go`, `leitura_atual_test.go`,
`caixa_de_saida_test.go` (drenagem por duas réplicas), `reservar_test.go`,
`performance_test.go`, e `planos_consulta_test.go` agora sobre o SQL emitido pelo GORM.

## 3. Banco já populado pela versão anterior (SC-004)

1. No commit anterior à troca: `docker compose up --build`, cadastrar sala e sessão
   (REST, `urls.txt`) e listar.
2. Na versão nova: `docker compose up --build catalogo` **sem** recriar o volume do Postgres.
3. Esperado: os mesmos filmes/salas/sessões aparecem, fatos pendentes são publicados, e nenhuma
   migração de dados rodou.

## 4. Fluxo ponta a ponta (SC-003)

`docker compose up --build`; seguir o quickstart da 001
(`specs/001-catalogo-sessoes-reserva/quickstart.md`): listar filmes → grade de sessões →
criar sala e sessão (confirmar `sessao.criada` no painel do RabbitMQ) → reservar.
Esperado: mesmos resultados e mesmas categorias de erro da versão anterior.
