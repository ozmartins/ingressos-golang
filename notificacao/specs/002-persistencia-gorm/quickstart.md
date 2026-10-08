# Quickstart: validar a equivalência após a troca para GORM

Pré-requisitos: Go 1.25+, Docker (Testcontainers e `docker compose`).

## 1. Núcleo e arquitetura (sem banco)

```
cd ingressos-golang/notificacao
make test
```

Esperado: tudo verde, inclusive `test/arquitetura_test.go` (o núcleo não importa
`gorm.io` nem `pgx`) e as suítes de domínio/casos de uso sem tocar em banco (US3).

## 2. Integração contra Postgres e RabbitMQ reais

```
make test-integration
```

Esperado: as mesmas suítes de antes passam sem asserção alterada (SC-001): emissão
idempotente e concorrente, baixa concorrente, listagem com filtro, aviso, schema próprio,
falha transitória → fila morta (SC-002).

## 3. Ponta a ponta no compose (SC-003, SC-004)

```
cd ingressos-golang && docker compose up --build
```

Com a plataforma de pé, publicar um `pagamento.sucesso` (`make publicar-pagamento` em
`notificacao/`), então listar em `GET /api/v1/ingressos/meus-ingressos` e validar via `POST /api/v1/ingressos/validar` o
ingresso. Repetir a publicação com a mesma reserva: continua existindo um único ingresso.

## 4. Banco já populado (US2)

Subir a versão anterior, emitir alguns ingressos, trocar a imagem do `notificacao` pela
nova (sem rodar migração nova) e confirmar que listagem, consulta e validação enxergam os
registros antigos — zero scripts de dados.
