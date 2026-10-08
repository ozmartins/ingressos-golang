# Quickstart: validar a troca para GORM

Pré-requisitos: Docker (Testcontainers e `docker compose`), Go 1.25+. Rodar de dentro
de `ingressos-golang/estoque/`.

## 1. Núcleo isolado (sem banco)

```
make test            # go test -race ./...
```

Esperado: tudo verde, incluindo `test/arquitetura_test.go` — que agora também proíbe
`gorm.io` no domínio e nos casos de uso (SC-005).

## 2. Equivalência com Postgres real (SC-001, SC-002)

```
make test-integration
```

Esperado: mesmas suítes de antes, asserções inalteradas — em especial
`invariante_test.go`/`bloqueio_test.go` (exatamente uma de N solicitações vence),
`desfechos_test.go`/`broker_test.go` (idempotência, reentrega), `expiracao_test.go`,
`esquema_test.go` (`search_path = estoque`, nada em `public`) e `desempenho_test.go`.

## 3. Banco já populado pela versão anterior (SC-004)

1. `git stash`/checkout do commit anterior à troca, `docker compose up --build`,
   publicar sessão e criar uma reserva (`make publicar-sessao`, chamada gRPC ou
   quickstart da 001).
2. Voltar para a versão nova, `docker compose up --build estoque` (sem recriar o
   volume do Postgres).
3. Esperado: reserva existente aparece no mapa, `publicar-pagamento` a confirma,
   nenhuma migração de dados rodou.

## 4. Fluxo ponta a ponta (SC-003)

`docker compose up --build`; seguir o quickstart da 001
(`specs/001-estoque-bloqueio-poltronas/quickstart.md`): `publicar-sessao` →
bloqueio via gRPC → `publicar-pagamento` (sucesso e falha) → expiração em 10 min.
Esperado: mesmos resultados e mesmas mensagens de erro da versão anterior.

## Linha de base (antes da troca, 2026-10-07)

- `make test`: tudo verde.
- `go test -race -tags=integration ./test/integration/...`: 31 testes, todos verdes (~32 s).
- `TestDesempenhoDoBloqueio`: n=500, mediana=1,30 ms, **p99=2,23 ms**.
- `TestDesempenhoDaConsulta` (500 lugares): mediana=5,46 ms, p99=6,56 ms.

## Verificação (2026-10-07, após a troca)

- `make test`: verde. `go test -race -tags=integration ./test/integration/...`: **32/32** verdes
  (os 31 de antes + `TestFalhaNoFatoDesfazTodaAConcessao`); concorrência/invariante com
  `-race -count=3`: verde.
- **Arquitetura (US3)**: `gorm.io` entrou na lista de proibidos ao núcleo. Prova negativa: um
  `import _ "gorm.io/gorm"` temporário em `internal/usecase` fez `TestNucleoNaoImportaInfraestrutura`
  falhar apontando o import; revertido.
- **Sem DDL (T017)**: `grep -rniE "AutoMigrate|CREATE TABLE|ALTER TABLE|Migrator\(" internal cmd` → vazio.
- **Banco populado pela versão anterior (T018, SC-004)**: o commit `fd6f9b8` (worktree destacado) provisionou
  uma sessão e criou 2 reservas (uma já vencida); mais uma reserva `CONFIRMADA` legada com `valor_total NULL`
  inserida à mão. A versão nova, sobre o mesmo banco e sem migração: leu o mapa (6 poltronas, estados certos),
  confirmou uma reserva (`aplicada`, e `ignorada-duplicata` na repetição), expirou a vencida, publicou os 2 fatos
  do outbox com `trace_context` intacto, e `CancelarPendentesDaSessao` contou as 2 confirmadas sem tocar nelas.
- **Do zero (T019)**: `docker compose up --build estoque` (projeto isolado, volumes removidos depois):
  `migrate-estoque` aplicou 1–3, o contêiner novo subiu e `/health/ready` respondeu
  `{"postgres":"ok","rabbitmq":"ok","redis":"ok"}`; `make publicar-sessao` provisionou a matriz via consumo.
  (Fluxo de bloqueio por gRPC/REST não exercitado nesse contêiner — exige mTLS/JWT; coberto pelos testes de integração.)
- `make lint`: **não executado** — `golangci-lint` não está instalado nesta máquina. `go vet` limpo.

### Latência (SC-006)

p99, 500 amostras, mesma máquina, Postgres 16 em contêiner:

| | bloqueio | consulta do mapa (500) |
|---|---|---|
| antes, `-race` (3 execuções) | 2,7 – 3,5 ms | 6,1 – 7,9 ms |
| depois, `-race` (4 execuções) | 4,2 – 9,9 ms | 9,2 – 13,6 ms |
| antes, sem `-race` (2) | 2,2 – 3,1 ms | 2,9 ms |
| depois, sem `-race` (4) | 2,1 – 2,8 ms | 2,6 – 3,6 ms |

Conclusão: **sem `-race` a latência é equivalente** (dentro do ruído); **com `-race` — que é como
`make test-integration` roda — fica ~50% acima** no bloqueio e na consulta. Ambos seguem muito abaixo
dos orçamentos absolutos (100 ms / 200 ms). O limite de "≤ 120% da linha de base" do SC-006 **não é atingido sob `-race`**;
ver relatório para a decisão do mantenedor. Uma otimização foi necessária e aplicada: `MapaDaSessao` passou de `Find`
em `[]poltronaRow` (p99 ~25 ms sob `-race`) para `Rows()` + `Scan` direto (p99 ~10 ms).
