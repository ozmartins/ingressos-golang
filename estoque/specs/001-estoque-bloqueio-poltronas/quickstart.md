# Quickstart — Validar o `Servico-Estoque` ponta a ponta

Guia de execução e validação. Não contém implementação: os detalhes de desenho
estão em [`data-model.md`](./data-model.md) e [`research.md`](./research.md), e o
contrato em [`contracts/`](./contracts/).

## Pré-requisitos

- Docker e Docker Compose
- Go 1.25+ (para rodar os testes fora do contêiner)
- [`grpcurl`](https://github.com/fullstorydev/grpcurl) para exercitar o contrato à mão

## 1. Subir a infraestrutura e o serviço

```bash
cd estoque
make certs        # gera CA de desenvolvimento + par do servidor e do cliente
docker compose up --build
```

O compose sobe, nesta ordem de dependência: `postgres` → `migrate` (aplica
`migrations/` e sai) → `redis` → `rabbitmq` → `estoque`. O serviço só inicia
depois que as migrações terminam com sucesso, então nunca encontra esquema pela
metade.

As migrações criam o schema `estoque` e todas as tabelas dentro dele; o serviço
fixa o `search_path` nesse schema, então nada do estoque fica em `public` além
da tabela de controle do próprio golang-migrate. Fora do compose, use
`make migrate-up`, que anexa `&search_path=estoque` à `DATABASE_URL` para manter
essa tabela de controle no lugar.

Portas no host: gRPC `50051`, administração (saúde e métricas) `8090`, RabbitMQ
management `15673`, AMQP `55672`, PostgreSQL `55433`, Redis `56379`.

As portas das dependências são deslocadas de propósito: a máquina de
desenvolvimento normalmente já tem um PostgreSQL, um Redis ou um RabbitMQ de
outro projeto ocupando as portas padrão.

Verificação de prontidão:

```bash
curl -fsS http://localhost:8090/health/ready && echo OK
```

## 2. Provisionar a matriz de poltronas de uma sessão

O provisionamento acontece pelo consumo de `sessao.criada` (FR-033). Como o
`Servico-Catalogo` ainda não publica esse fato, publique-o à mão:

```bash
make publicar-sessao SESSAO=f781a9b2-11e2-4f81-a901-8890bc123456 FILEIRAS=A,B ASSENTOS=10
```

O alvo publica em `cinema.eventos` com routing key `sessao.criada` o payload
descrito em [`contracts/eventos.md`](./contracts/eventos.md).

**Esperado**: em segundos, 20 poltronas `LIVRE`. Confira pelo próprio contrato:

```bash
grpcurl -cacert certs/ca.pem -cert certs/cliente.pem -key certs/cliente-key.pem \
  -d '{"sessao_id":"f781a9b2-11e2-4f81-a901-8890bc123456"}' \
  localhost:50051 estoque.ServicoEstoque/ConsultarMapaPoltronas
```

Republicar o mesmo evento não deve duplicar nada (FR-034) — repita o comando
acima e confirme que a contagem continua 20.

## 3. Bloquear poltronas

```bash
grpcurl -cacert certs/ca.pem -cert certs/cliente.pem -key certs/cliente-key.pem \
  -d '{"sessao_id":"f781a9b2-...","poltronas_ids":["A1","A2"],"usuario_id":"c394c8b3-..."}' \
  localhost:50051 estoque.ServicoEstoque/BloquearPoltronas
```

**Esperado**: `sucesso: true`, `reserva_id` preenchido e `expira_em` ≈ agora + 10
min (FR-001, FR-007). O mapa passa a mostrar `A1` e `A2` como `RESERVADA`, e a
fila de quem escuta `reserva.criada` recebe o evento (veja em
http://localhost:15673, usuário `guest`/`guest`).

Repetir o mesmo comando deve devolver `sucesso: false` com
`motivo: POLTRONAS_INDISPONIVEIS` (FR-002).

### Recusas que devem ser distinguíveis (FR-046, `contracts/erros.md`)

| Tentativa | Esperado |
|---|---|
| `poltronas_ids: []` | `INVALID_ARGUMENT` |
| `["A1","A1"]` | `INVALID_ARGUMENT` |
| 11 rótulos (limite padrão 10) | `INVALID_ARGUMENT`, `reason=LIMITE_POLTRONAS_EXCEDIDO` |
| `["Z9"]` numa sessão que não a tem | `FAILED_PRECONDITION` |
| sessão nunca provisionada | `FAILED_PRECONDITION` |
| `ConsultarMapaPoltronas` de sessão desconhecida | `NOT_FOUND` |
| chamada **sem** `-cert`/`-key` | handshake TLS recusado (FR-037) |

## 4. Confirmar o pagamento

```bash
make publicar-pagamento RESERVA=<reserva_id> RESULTADO=sucesso
```

**Esperado**: reserva `CONFIRMADA`, `A1` e `A2` como `OCUPADA` (FR-019), e o
prazo de 10 minutos passa a não ter efeito sobre elas (FR-014).

Publicar o **mesmo** evento de novo não deve mudar nada e deve ser confirmado
normalmente (FR-021) — verifique que a contagem de `OCUPADA` continua 2.

## 5. Falha de pagamento libera as poltronas

Bloqueie `B1`, depois:

```bash
make publicar-pagamento RESERVA=<reserva_id> RESULTADO=falhou
```

**Esperado**: reserva `CANCELADA` e `B1` de volta a `LIVRE` (FR-020), imediatamente
bloqueável por outra pessoa.

## 6. Expiração automática

```bash
docker compose exec estoque env RESERVA_TTL=20s  # ou suba o serviço com esse valor
```

Bloqueie `B2`, não publique desfecho e aguarde ~25 s.

**Esperado**: reserva `EXPIRADA` e `B2` de volta a `LIVRE` sem qualquer ação
(FR-012). Para provar o caminho autoritativo (FR-013, SC-008), pare o Redis
(`docker compose stop redis`) antes do prazo: a liberação continua acontecendo,
apenas pela varredura periódica.

## 7. Suíte automatizada

```bash
make test              # unitários e casos de uso — sem infraestrutura
make test-integration  # Testcontainers: Postgres + Redis + RabbitMQ reais
```

A suíte de integração cobre os critérios que não se verificam à mão:

| Critério | O que o teste faz |
|---|---|
| SC-002 | 100 solicitações paralelas sobre o mesmo conjunto; exatamente uma concedida |
| SC-004 | Reentrega duplicada de `pagamento.sucesso` e de `sessao.criada` |
| SC-005 | Broker derrubado no instante da concessão; evento chega depois |
| SC-006 | 1.000 ciclos aleatórios; nenhuma poltrona em estado inconsistente |
| SC-008 | Serviço parado além do prazo; reservas vencidas invalidadas no retorno |
| SC-011 | Chamador sem certificado, com certificado expirado e de CA desconhecida |
| SC-012 | Banco indisponível; toda solicitação recusada, nenhuma reserva criada |

## 8. Derrubar

```bash
docker compose down -v
```
