# Servico-Estoque

Serviço de gestão de assentos e controle de concorrência do sistema de cinema.
É o **dono do estado de disponibilidade das poltronas**: nenhum outro serviço
decide se um assento pode ser vendido.

- Expõe uma **API REST** autenticada por JWT do Keycloak com duas operações —
  reservar (bloquear) poltronas de forma atômica e consultar o mapa de uma
  sessão. A documentação interativa fica em `/docs`.
- Para reservar, consulta a sessão no **Servico-Catalogo** (`GET /api/v1/sessoes/{id}`,
  com timeout e circuit breaker), recusa se ela não aceita reserva e calcula o
  valor como `preco_base × quantidade`.
- Reage a três fatos do **RabbitMQ**: `sessao.criada` (provisiona a matriz),
  `pagamento.sucesso` (confirma) e `pagamento.falhou` (libera).
- Publica `reserva.criada` a cada bloqueio concedido.
- Invalida sozinho as reservas não pagas em 10 minutos.

## As três decisões que moldam o desenho

1. **A exclusividade mora no PostgreSQL.** O bloqueio trava as linhas das
   poltronas com `SELECT ... FOR UPDATE NOWAIT`, em ordem determinística de
   rótulo, na mesma transação que grava reserva, vínculos e o fato. O Redis, que
   a ERS pede, ficou com o índice de prazo: **perdê-lo inteiro atrasa a
   liberação, nunca permite venda dupla.**
2. **Caixa de saída transacional.** O fato `reserva.criada` é persistido junto
   com a reserva e publicado depois, fora do caminho da requisição. Sobrevive a
   um broker fora do ar sem duplicar reserva e sem gastar o orçamento de 100 ms.
3. **Idempotência em duas camadas.** Registro de mensagem processada mais guarda
   de máquina de estados (`WHERE status = 'PENDENTE'`). Duplicata e chegada fora
   de ordem são inofensivas por construção.

## API REST

```
POST /api/v1/sessoes/{sessao_id}/bloqueios   → 201 { reserva_id, expira_em, mensagem }
GET  /api/v1/sessoes/{sessao_id}/poltronas   → 200 { sessao_id, poltronas[] }
```

O corpo do bloqueio é só `{"poltronas_ids": ["A1", "A2"]}`: campos desconhecidos
são recusados. O `usuario_id` vem da claim `sub` (assinada) e o valor, do preço
da sessão no catálogo — nenhum dos dois é aceito do chamador. Um teste trava essa
regressão.

Os erros seguem a RFC 9457 (`application/problem+json`); o último segmento do
`type` é a categoria estável. Entre elas: `poltronas-indisponiveis` (409),
`sessao-nao-encontrada` (404), `sessao-nao-reservavel` (422),
`catalogo-indisponivel` (503) e `resposta-invalida-do-parceiro` (502).

## Documentação da API

`/openapi.yaml` devolve o contrato REST e `/docs` abre o Swagger UI sobre ele
(http://localhost:8085/docs no compose). Nenhum dos dois exige credencial: pedir
token para ler o contrato barraria justamente quem ainda vai integrar.

O contrato fica em `internal/adapter/http/openapi/openapi.yaml`, embutido no binário
com `go:embed` e servido como está.

## Subir localmente

```bash
docker compose -f ../docker-compose.yml up -d --build estoque
curl -fsS http://localhost:8090/health/ready
```

As tabelas do estoque vivem no schema `estoque`, não em `public`: as migrações
criam o schema e qualificam cada objeto, e o serviço fixa o `search_path` no
pool de conexões. Para inspecionar o banco com `psql`, aponte o `search_path`
antes (`SET search_path TO estoque;`) ou qualifique as tabelas. A tabela de
controle do golang-migrate (`schema_migrations`) mora no mesmo schema: em
`public` os quatro serviços disputariam uma só.

Portas no host: REST `8085`, administração `8090`, PostgreSQL `55433`,
Redis `56379`, AMQP `55672`, RabbitMQ management `15673`. As portas das
dependências são deslocadas de propósito, para não colidir com outros projetos.

## Testes

```bash
make test              # domínio, casos de uso, adaptadores HTTP e arquitetura — sem infraestrutura
make test-integration  # Postgres, Redis e RabbitMQ reais via Testcontainers
```

A suíte de integração é onde os critérios caros vivem: exatamente um vencedor
entre 100 solicitações concorrentes, reentrega sem efeito adicional, expiração
após parada do serviço, correlação ponta a ponta atravessando o broker e mil
ciclos aleatórios sem estado inconsistente.

## Configuração

Toda por ambiente; o processo **recusa subir** com variável obrigatória ausente
ou malformada.

| Variável | Padrão | Para quê |
|---|---|---|
| `DATABASE_URL` | — (obrigatória) | PostgreSQL, fonte de verdade (o `search_path` é fixado no código, no schema `estoque`) |
| `RABBITMQ_URL` | — (obrigatória) | barramento de fatos |
| `REDIS_URL` | vazio | índice de prazo; ausente, só a varredura expira |
| `ADMIN_ADDR` | `:8090` | saúde e métricas |
| `HTTP_ADDR` | `:8085` | API REST e `/docs` |
| `JWKS_URL` | — (obrigatória) | chaves do Keycloak, carregadas na largada, que validam o JWT da API REST |
| `JWT_ISSUER` | — (obrigatória) | emissor aceito no token |
| `JWT_AUDIENCE` | — (obrigatória) | público aceito no token |
| `CATALOGO_URL` | — (obrigatória) | base HTTP do Servico-Catalogo, de onde vêm preço e status da sessão |
| `CATALOGO_TIMEOUT` | `2s` | tempo máximo de cada consulta ao catálogo |
| `BREAKER_FALHAS_CONSECUTIVAS` | `5` | falhas seguidas que abrem o circuit breaker do catálogo |
| `BREAKER_INTERVALO_ABERTO` | `30s` | quanto o breaker fica aberto antes de testar de novo |
| `RESERVA_TTL` | `10m` | prazo da reserva |
| `POLTRONAS_MAX_POR_BLOQUEIO` | `10` | teto por solicitação |
| `VARREDURA_EXPIRACAO_INTERVALO` | `10s` | intervalo da varredura autoritativa |
| `AMQP_PREFETCH` | `32` | trabalho não confirmado por consumidor |
| `RETENCAO_MENSAGENS_PROCESSADAS` | `720h` | janela de idempotência |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | vazio | coletor de métricas e rastros |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

## Estrutura

```text
cmd/estoque/       composição única do binário
cmd/publicar/      ferramenta de linha de comando para publicar fatos à mão
internal/domain/   núcleo: poltrona, reserva, invariantes — sem infraestrutura
internal/usecase/  orquestração e portas
internal/adapter/  HTTP (API e cliente do catálogo), PostgreSQL, AMQP, Redis
internal/platform/ configuração, observabilidade, saúde
migrations/        esquema versionado
test/              integração (Testcontainers), arquitetura
```

`internal/domain` e `internal/usecase` não importam adaptador nenhum. Isso é
verificado por teste (`test/arquitetura_test.go`) e pelo linter, não por revisão.

## Pendências de integração

- O `Servico-Catalogo` publica `sessao.criada` desde que passou a guardar a planta
  das salas, e o contrato do fato foi adotado por ele sem alteração. O ciclo de vida está fechado: ele também
  publica `sessao.cancelada`, que este serviço consome para soltar as reservas
  pendentes, e `sessao.alterada`, que não afetamos porque a sala de uma sessão
  não pode mudar do lado dele.
- O valor da reserva (`valor_total`) é calculado aqui, a partir do `preco_base`
  da sessão lido no catálogo; ele é guardado na reserva e repassado em
  `reserva.criada`, que é o que o `Servico-Pagamento` cobra.
