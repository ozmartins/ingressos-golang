# Servico-Estoque

Serviço de gestão de assentos e controle de concorrência do sistema de cinema.
É o **dono do estado de disponibilidade das poltronas**: nenhum outro serviço
decide se um assento pode ser vendido.

- Expõe duas operações — bloquear poltronas de forma atômica e consultar o mapa
  de uma sessão — em **duas superfícies**: um canal **gRPC** autenticado por
  mTLS, para os serviços, e uma **API REST** autenticada por JWT do Keycloak,
  para o cliente final. As duas chamam os mesmos casos de uso, então valem as
  mesmas regras; a documentação interativa da REST fica em `/docs`.
- Reage a três fatos do **RabbitMQ**: `sessao.criada` (provisiona a matriz),
  `pagamento.sucesso` (confirma) e `pagamento.falhou` (libera).
- Publica `reserva.criada` a cada bloqueio concedido.
- Invalida sozinho as reservas não pagas em 10 minutos.

## Como isso foi construído

A especificação, o plano, o modelo de dados, os contratos e a lista de tarefas
estão em [`specs/001-estoque-bloqueio-poltronas/`](specs/001-estoque-bloqueio-poltronas/),
produzidos pelo fluxo spec-kit (`specify` → `clarify` → `plan` → `tasks` →
`analyze` → `implement`). As regras que governam o projeto estão em
[`.specify/memory/constitution.md`](.specify/memory/constitution.md) — o plano
tem uma seção de verificação contra cada princípio.

Quatro dessas regras valem também para quem só vai ler ou mexer no código: nada de
complexidade além da necessária ou pedida (VII); domínio e operações expostas têm
teste automatizado (VIII); **o código é a fonte da verdade, não a spec** (IX) — os
documentos acima são instrumentos de projeto, e afirmação sobre comportamento atual
se verifica no código; e divergência entre código e spec é pergunta ao mantenedor,
não decisão de quem encontrou (X).

Leia primeiro, nesta ordem: [`spec.md`](specs/001-estoque-bloqueio-poltronas/spec.md)
(o quê e por quê), [`research.md`](specs/001-estoque-bloqueio-poltronas/research.md)
(as decisões técnicas e o que foi rejeitado),
[`data-model.md`](specs/001-estoque-bloqueio-poltronas/data-model.md)
(o protocolo transacional do bloqueio).

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

## As duas superfícies

As mesmas duas operações são oferecidas em gRPC e em REST. Ambas chamam os
mesmos casos de uso — este serviço não tem regra que valha em um transporte e
não no outro. O que muda é quem chama:

| | gRPC (`:50051`) | REST (`:8085`) |
|---|---|---|
| Chamador | outros serviços | cliente final |
| Identidade | mTLS | JWT do Keycloak |
| `usuario_id` | vem no corpo, porque o Servico-Catalogo já validou o token | vem da claim `sub`, que é assinada |
| Poltronas indisponíveis | `OK` com `sucesso=false` | `409` |
| Prazo da reserva | epoch em segundos | RFC 3339 |

```
POST /api/v1/sessoes/{sessao_id}/bloqueios   → 201 { reserva_id, expira_em, mensagem }
GET  /api/v1/sessoes/{sessao_id}/poltronas   → 200 { sessao_id, poltronas[] }
```

O `usuario_id` **não** é lido do corpo na superfície REST: se fosse, qualquer
chamador reservaria em nome de outra pessoa. Um teste trava essa regressão.

Os erros seguem a RFC 9457 (`application/problem+json`). Cada `type` corresponde
a uma razão já publicada em [`contracts/erros.md`](specs/001-estoque-bloqueio-poltronas/contracts/erros.md),
com o status HTTP equivalente ao código gRPC: `InvalidArgument`→400,
`FailedPrecondition`→422, `NotFound`→404, `Unavailable`→503, `Internal`→500.

## Documentação da API

`/openapi.yaml` devolve o contrato REST e `/docs` abre o Swagger UI sobre ele
(http://localhost:8085/docs no compose). Nenhum dos dois exige credencial: pedir
token para ler o contrato barraria justamente quem ainda vai integrar.

O contrato versionado em `specs/001-estoque-bloqueio-poltronas/contracts/openapi.yaml`
é a fonte da verdade; a cópia embutida no binário é gerada por `make openapi-sync`,
e um teste de paridade falha quando as duas divergem. O contrato gRPC continua
sendo o `.proto` no mesmo diretório.

## Subir localmente

```bash
make certs           # CA e pares de desenvolvimento (descartáveis)
docker compose -f ../docker-compose.yml up -d --build estoque
curl -fsS http://localhost:8090/health/ready
```

As tabelas do estoque vivem no schema `estoque`, não em `public`: as migrações
criam o schema e qualificam cada objeto, e o serviço fixa o `search_path` no
pool de conexões. Para inspecionar o banco com `psql`, aponte o `search_path`
antes (`SET search_path TO estoque;`) ou qualifique as tabelas. A tabela de
controle do golang-migrate (`schema_migrations`) mora no mesmo schema: em
`public` os quatro serviços disputariam uma só.

O roteiro completo de validação — provisionar sessão, bloquear, confirmar,
liberar, expirar — está em
[`quickstart.md`](specs/001-estoque-bloqueio-poltronas/quickstart.md).

Portas no host: gRPC `50051`, administração `8090`, PostgreSQL `55433`,
Redis `56379`, AMQP `55672`, RabbitMQ management `15673`. As portas das
dependências são deslocadas de propósito, para não colidir com outros projetos.

## Testes

```bash
make test              # domínio, casos de uso e contrato gRPC — sem infraestrutura
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
| `GRPC_ADDR` | `:50051` | canal síncrono entre serviços |
| `ADMIN_ADDR` | `:8090` | saúde e métricas |
| `HTTP_ADDR` | `:8085` | API REST e `/docs` |
| `JWKS_URL` | — (obrigatória) | chaves do Keycloak, carregadas na largada, que validam o JWT da API REST |
| `JWT_ISSUER` | — (obrigatória) | emissor aceito no token |
| `JWT_AUDIENCE` | — (obrigatória) | público aceito no token |
| `TLS_CLIENT_AUTH` | `require` | `require` exige identidade de serviço; `off` só em desenvolvimento |
| `TLS_CERT_FILE`, `TLS_KEY_FILE`, `TLS_CLIENT_CA_FILE` | — | material de identidade (obrigatórios com `require`) |
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
internal/adapter/  gRPC, PostgreSQL, AMQP, Redis
internal/platform/ configuração, observabilidade, saúde
migrations/        esquema versionado
test/              contrato (bufconn + mTLS), integração (Testcontainers), arquitetura
```

`internal/domain` e `internal/usecase` não importam adaptador nenhum. Isso é
verificado por teste (`test/arquitetura_test.go`) e pelo linter, não por revisão.

## Pendências de integração

- O `Servico-Catalogo` ainda não publica `sessao.criada`; até lá, a matriz é
  provisionada com `make publicar-sessao`. O contrato proposto está em
  [`contracts/eventos.md`](specs/001-estoque-bloqueio-poltronas/contracts/eventos.md).
- O `Servico-Catalogo` ainda disca em texto claro. Ativar `TLS_CLIENT_AUTH=require`
  em produção exige que ele passe a apresentar certificado de cliente.
