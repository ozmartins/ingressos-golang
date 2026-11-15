# Ingressos

Plataforma de venda de ingressos de cinema em quatro microsserviços Go, com
PostgreSQL, Redis, RabbitMQ e Keycloak, orquestrados por um único
[`docker-compose.yml`](docker-compose.yml) na raiz.

## Serviços

| Serviço | Papel | Superfícies |
|---|---|---|
| [`catalogo/`](catalogo/README.md) | Navegação e cadastro de filmes, cinemas, salas e sessões; dono do preço da sessão | REST (JWT), produtor AMQP |
| [`estoque/`](estoque/README.md) | Dono da disponibilidade das poltronas; reserva (consultando preço e status da sessão no catálogo), confirma e libera assentos | REST (JWT), cliente REST do catálogo, consumidor/produtor AMQP |
| [`pagamento/`](pagamento/README.md) | Cobra de forma assíncrona a reserva criada | REST (JWT), consumidor/produtor AMQP |
| [`notificacao/`](notificacao/README.md) | Emite o ingresso digital e valida na portaria | REST (JWT e `X-API-Key`), consumidor AMQP |

O fluxo assíncrono é `reserva.criada` → `pagamento.sucesso` / `pagamento.falhou`
→ emissão do ingresso.

## Papel de cada componente e como se comunicam

Esta seção foi escrita a partir do código-fonte (rotas, casos de uso, adaptadores
AMQP/HTTP e clientes do front), não só das specs.

### Visão geral

```
                      ┌──────────────┐  login (OIDC + PKCE)   ┌──────────┐
   navegador ────────▶│ front / admin│───────────────────────▶│ Keycloak │
                      └──────┬───────┘                        └────┬─────┘
          REST + Bearer JWT  │                    JWKS (valida JWT)│ (todas as APIs)
        ┌────────────────────┼─────────────────┐                   │
        ▼                    ▼                 │                   │
  ┌──────────┐  GET /sessoes/{id}  ┌──────────┐│
  │ catalogo │◀────────────────────│ estoque  ││
  └────┬─────┘      (REST)         └──┬───▲───┘│
       │ sessao.criada / .cancelada    │   │ pagamento.sucesso / .falhou
       └──────────────▶ RabbitMQ ◀─────┘   │
                  (exchange cinema.eventos)│
       reserva.criada ──▶ ┌───────────┐ ───┘
                          │ pagamento │── pagamento.sucesso ──▶ ┌─────────────┐
                          └───────────┘                        │ notificacao │
                                                               └─────────────┘
```

Há dois estilos de comunicação, escolhidos por finalidade:

- **REST síncrono** quando quem chama precisa da resposta *agora* para decidir:
  o navegador consultando/gravando, e o `estoque` lendo preço e status da sessão
  no `catalogo`.
- **Eventos assíncronos no RabbitMQ** (exchange `topic` `cinema.eventos`) quando
  um serviço apenas *anuncia um fato* e outros reagem no seu tempo. Nenhum
  serviço chama `pagamento` ou `notificacao` por HTTP.

### `catalogo` — o que existe e quanto custa

- **Papel:** fonte de verdade de filmes, cinemas, salas e sessões (horário, idioma,
  **preço base**, status). É dono do layout de poltronas de cada sala.
- **REST** (`/api/v1/filmes|cinemas|salas|sessoes`, CRUD): leituras são
  **públicas**; `POST/PUT/DELETE` exigem JWT do Keycloak.
- **Publica** (via caixa de saída transacional): `sessao.criada` (com o layout de
  poltronas da sala), `sessao.alterada` e `sessao.cancelada` (sessão removida).
  Hoje só `estoque` consome `criada` e `cancelada`; `sessao.alterada` não tem
  consumidor.
- **Para quê:** separar o cadastro (muda pouco, lido muito) da venda. Publicar o
  layout no evento evita que o `estoque` precise consultar o catálogo para montar
  o mapa de assentos.

### `estoque` — quem pode sentar onde

- **Papel:** dono da disponibilidade de cada poltrona (livre → reservada →
  ocupada). É a única peça que garante que o mesmo assento não é vendido duas
  vezes (exclusão com `SELECT ... FOR UPDATE NOWAIT` no PostgreSQL; o Redis guarda
  só o índice de expiração).
- **REST** (JWT): `GET /api/v1/sessoes/{id}/poltronas` (mapa com status) e
  `POST /api/v1/sessoes/{id}/bloqueios` (reserva temporária; devolve `reserva_id`
  e `expira_em`, 10 min por padrão, no máximo 10 poltronas).
- **Chama o `catalogo` por REST** (`GET /api/v1/sessoes/{id}`, rota pública, com
  timeout e disjuntor) para confirmar que a sessão aceita reservas e calcular o
  `valor_total`. Assim o preço nunca vem do cliente.
- **Consome:** `sessao.criada` (cria a matriz de poltronas da sessão),
  `sessao.cancelada` (cancela as reservas pendentes), `pagamento.sucesso`
  (reserva → confirmada) e `pagamento.falhou` (libera as poltronas).
- **Publica:** `reserva.criada`, gravada na **mesma transação** da reserva
  (outbox), então a reserva e o anúncio nunca divergem.
- **Expira** sozinho reservas não pagas no prazo (varredura periódica + índice no
  Redis); perder o Redis só atrasa a liberação.

### `pagamento` — cobrar a reserva

- **Papel:** transforma uma reserva em uma cobrança e anuncia o resultado.
- **Consome** `reserva.criada`: cria a transação em `AGUARDANDO_FORMA`, com valor,
  dono (`usuario_id`) e prazo igual ao da reserva.
- **REST** (JWT; só o dono da reserva enxerga/age): `GET
  /api/v1/pagamentos/reserva/{reserva_id}` consulta o estado e `POST` na mesma
  rota escolhe a forma (`PIX` ou `CARTAO_CREDITO`). Escolhida a forma, uma
  varredura aciona o adquirente, que hoje é **simulado** (centavos `.13` recusa,
  `.66` saldo insuficiente, `.99` trava, demais aprovam).
- **Publica:** `pagamento.sucesso` ou `pagamento.falhou` (inclusive por reserva
  expirada antes da cobrança).
- **Para quê:** isolar o meio de pagamento (instável por natureza) do estoque; a
  reserva já segurou o assento, então a cobrança pode ser lenta, repetida ou falhar
  sem afetar a venda de outras sessões.

### `notificacao` — o ingresso

- **Papel:** emite e valida o ingresso digital.
- **Consome** `pagamento.sucesso`: cria o ingresso (código assinado por HMAC com
  `INGRESSO_QR_SEGREDO`; idempotente por reserva) e dispara o aviso ao comprador
  (canal e-mail, hoje simulado).
- **REST:** `GET /api/v1/ingressos/meus-ingressos` (JWT do comprador, filtro por
  status) e `POST /api/v1/ingressos/validar` (**`X-API-Key`** do dispositivo da
  portaria; marca como utilizado e responde 200 autorizada, 409 reuso/inválido,
  404 desconhecido).
- **Para quê:** a emissão é consequência do pagamento, não parte dele; se o aviso
  falhar, o ingresso já está salvo e a venda não é desfeita.

### Fluxo ponta a ponta

1. O usuário escolhe filme, cinema e sessão (`front` → `catalogo`, público).
2. Faz login no Keycloak e abre o mapa (`front` → `estoque`, JWT).
3. Seleciona assentos: `POST .../bloqueios`. O `estoque` consulta o `catalogo`,
   trava as poltronas e grava `reserva.criada` na caixa de saída.
4. `pagamento` consome `reserva.criada` e abre a transação.
5. A forma de pagamento é escolhida (`POST /pagamentos/reserva/{id}`) e a cobrança
   roda → `pagamento.sucesso` ou `pagamento.falhou`.
6. `estoque` confirma ou libera os assentos; em `sucesso`, `notificacao` emite o
   ingresso, consultável em `meus-ingressos` e validado na portaria.
7. Sem pagamento em 10 min, o `estoque` expira a reserva por conta própria.

### Garantias de entrega das mensagens

- **Outbox transacional** em `catalogo`, `estoque` (e anúncios de `pagamento`
  persistidos antes de publicar): o fato só sai se a mudança de estado foi gravada.
- **Consumidores idempotentes** (tabela de mensagens processadas e guardas de
  estado `WHERE status = ...`): reentrega do broker não duplica efeito.
- **Filas com fila morta (DLQ)** por consumidor: mensagem inválida ou que estoura o
  limite de entregas vai para `<fila>.dlq` em vez de travar a fila.
- **Autenticação:** todas as APIs validam o JWT do realm `cinema` pelo JWKS do
  Keycloak (`sub` vira o `usuario_id`); o `trace_id` atravessa REST e AMQP
  (OpenTelemetry → Jaeger; métricas em Prometheus).

### Frontends

Ambos são SPAs React + Vite, falam só com a API por REST e usam o Keycloak
(Authorization Code + PKCE, client `cinema-app`). Como as APIs não enviam CORS,
o servidor do Vite faz proxy (`/catalogo` → `:8082`, `/estoque` → `:8085`,
`/identidade` → `:8081` no admin).

- **[`front/`](front/README.md) — vitrine do público.** Lista filmes e cinemas,
  mostra a programação e os cinemas de cada filme (`catalogo`, sem login), exige
  login para ver o mapa de assentos e reservar (`estoque`) e termina numa página
  de pagamento com cartão e no PDF do ingresso gerado no navegador (`jspdf`).
- **[`admin/`](admin/README.md) — terminal retrô de administração.** Comandos
  (`filmes|cinemas|salas|sessoes listar|obter|criar|atualizar|remover`, `health`,
  `login`) que operam o CRUD do `catalogo` com token Keycloak. É o caminho para
  cadastrar sessões, o que por sua vez dispara `sessao.criada` e provisiona as
  poltronas no `estoque`.

> **Limite atual do front:** ele **não chama** `pagamento` nem `notificacao`
> (não há proxy nem cliente para eles). A tela de pagamento é uma simulação local
> (temporizador; número de cartão terminado em `0000` é recusado) e o PDF é montado
> no cliente. Na prática, a reserva criada pelo front gera `reserva.criada` e o
> `pagamento` abre a transação, mas ela fica em `AGUARDANDO_FORMA` até o prazo e
> a reserva expira no `estoque`; nenhum ingresso é emitido por esse caminho. As
> etapas 5 e 6 acima funcionam hoje por chamada direta às APIs (Swagger/curl) ou
> pelos scripts `publicar` de cada serviço.

## Persistência: um banco, um schema por serviço

Os quatro serviços usam a mesma instância PostgreSQL e o mesmo banco (`cinema`),
mas **nenhum compartilha tabelas com outro**. Foi uma escolha para manter a
infraestrutura local simples (um único contêiner de banco); o isolamento é
feito por schema e por papel:

| Serviço | Schema | Papel (login) |
|---|---|---|
| `catalogo` | `catalogo` | `catalogo` |
| `estoque` | `estoque` | `estoque` |
| `pagamento` | `pagamento` | `pagamento` |
| `notificacao` | `notificacao` | `notificacao` |

- [`infra/postgres/init/01-papeis-e-schemas.sql`](infra/postgres/init/01-papeis-e-schemas.sql)
  cria cada schema com `AUTHORIZATION` do próprio papel, sem nenhum `GRANT`
  cruzado, e revoga `CREATE` no schema `public`. Um serviço não tem permissão
  para ler nem escrever no schema de outro.
- Cada serviço fixa o `search_path` no seu schema, tanto em runtime
  (`internal/adapter/postgres`) quanto nas migrations (a tabela
  `schema_migrations` também fica no schema do serviço).
- Nenhuma query ou migration referencia o schema de outro serviço. A troca de
  dados entre serviços acontece só por REST e por mensagens AMQP.
- Separar em bancos físicos distintos exigiria apenas trocar a `DATABASE_URL` de
  cada serviço no `docker-compose.yml`; o código não muda.

## Subir tudo

```bash
docker compose up --build
```

As portas do host e os endereços de cada serviço, incluindo os `/docs` (Swagger
UI), os health checks, o painel do RabbitMQ e o console do Keycloak, estão em
[`urls.txt`](urls.txt). Todas as portas são configuráveis pelas variáveis
`PORTA_*` do compose.
