# Quickstart: validar a equivalência após a troca para GORM

Pré-requisitos: Go 1.25+, Docker (Testcontainers e `docker compose`).

## 1. Núcleo e arquitetura (sem banco)

```
cd ingressos-golang/pagamento
make test
make lint
```

Esperado: `make test` verde (domínio e casos de uso sem tocar em banco) e `make lint` sem
nenhum achado do `depguard` — o núcleo não importa `gorm.io` nem `pgx` (US3). Os demais
achados do linter, se houver, são anteriores a esta feature.

## 2. Integração contra Postgres e RabbitMQ reais

```
make test-integration
```

Esperado: as mesmas suítes de antes passam sem asserção alterada (SC-001): registro
idempotente e concorrente, reivindicação concorrente, varredura de cobranças e de esperas
vencidas, anúncio do resultado, schema próprio (SC-002).

## 3. Ponta a ponta no compose (SC-003, SC-004)

```
cd ingressos-golang && docker compose up --build
```

Com a plataforma de pé, seguir o quickstart da spec 001
(`specs/001-pagamento-assincrono/quickstart.md`): anunciar uma reserva
(`make publicar-reserva RESERVA=... VALOR=... FORMA=... EXPIRA_EM=...`), escolher a forma
de pagamento e consultar o andamento em
`GET /api/v1/pagamentos/reserva/{reserva_id}`; conferir `pagamento.sucesso`/`pagamento.falhou`
na fila (`make espiar-evento ROUTING=pagamento.sucesso`). Repetir o anúncio com a mesma reserva: continua existindo uma única transação.

## 4. Banco já populado (US2)

Subir a versão anterior, gerar algumas transações (pagas, recusadas, aguardando forma),
trocar a imagem do `pagamento` pela nova (sem rodar migração nova) e confirmar que
consulta, varredura e anúncio enxergam os registros antigos — zero scripts de dados.
