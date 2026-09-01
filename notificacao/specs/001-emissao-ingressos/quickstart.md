# Quickstart: Emissão e Validação de Ingressos Digitais

**Fase 1** | **Data**: 2026-08-30

Roteiro de validação ponta a ponta. Prova, com o serviço rodando de verdade, que um
pagamento confirmado vira ingresso, que o ingresso aparece para a pessoa e que a
portaria o valida uma vez só.

> **Estado deste roteiro — executado em 2026-08-30**, contra PostgreSQL 16 e RabbitMQ
> 3.13 reais, com o binário compilado deste repositório.
>
> **Verificado em execução**: sanidade da configuração (o processo recusa subir sem
> `INGRESSO_QR_SEGREDO`, e recusa `AMQP_LIMITE_ENTREGAS=abc` em vez de cair no padrão) e
> os cenários **1 a 8**, com os desfechos que cada um descreve.
>
> **Não verificado por este roteiro**: a segunda metade do cenário 7 — a falha
> transitória com o banco derrubado — foi exercitada pelo teste de integração
> `test/integration/emissao_test.go`, e não à mão. Os três critérios de latência não são
> medidos aqui (ver o mapa no fim do arquivo).
>
> **Desvios do ambiente desta execução**, que valem para quem for repetir: as portas
> 5432, 5433 e 8080 estavam ocupadas na máquina, então o Postgres subiu em 5434
> (`PORTA_POSTGRES=5434`) e o serviço em 8099 (`PORTA_HTTP=8099`); a migração foi
> aplicada via `psql` dentro do contêiner, porque o binário `migrate` não estava
> instalado; e o cenário 3 exigiu um servidor JWKS estático de mentira, porque não há
> Keycloak nesta máquina e o serviço busca o conjunto de chaves na largada.

---

## Pré-requisitos

- Go 1.27 e Docker (com Compose)
- `make`
- `migrate` (golang-migrate) no `PATH` — ou aplique a migração com
  `docker compose exec -T postgres psql -U postgres -d notificacao -f - < migrations/000001_criar_ingressos.up.sql`
- um emissor de identidade que sirva JWKS, para o cenário 3. Sem Keycloak à mão, sirva um
  JWKS estático e assine o token com a mesma chave; sem isso o processo **não sobe**,
  porque busca o conjunto de chaves na largada

## Ambiente

```bash
cd notificacao
docker compose up -d          # Postgres + RabbitMQ
# PORTA_POSTGRES permite desviar de um PostgreSQL local já ocupando a 5432.
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/notificacao?sslmode=disable'
make migrate-up
```

A migração cria o schema `notificacao` e as duas tabelas dentro dele; o serviço
fixa o `search_path` nesse schema no próprio pool, então nada da notificação
fica em `public` além da tabela de controle do golang-migrate — `make
migrate-up` anexa `&search_path=public` à `DATABASE_URL` justamente para manter
essa tabela no lugar. Os `psql` deste roteiro qualificam as tabelas
(`notificacao.ingressos_emitidos`) porque a sessão do `psql` não herda esse
`search_path`.

Variáveis do processo:

```bash
export AMQP_URL='amqp://guest:guest@localhost:5672/'
export INGRESSO_QR_SEGREDO='segredo-de-desenvolvimento-nao-usar-em-producao'
export PORTARIA_API_KEY='chave-de-portaria-de-teste'
export JWKS_URL='http://localhost:8081/realms/cinema/protocol/openid-connect/certs'
export JWT_ISSUER='http://localhost:8081/realms/cinema'
export JWT_AUDIENCE='conta-cinema'
```

Sanidade da configuração — apague uma variável obrigatória e confirme que o processo
**recusa subir** listando o que falta (D11, `contracts/erros.md` §5):

```bash
env -u INGRESSO_QR_SEGREDO make run    # esperado: erro de configuração e saída != 0
make run                               # esperado: sobe e declara a topologia
```

---

## Cenário 1 — Pagamento confirmado vira ingresso (P1, FR-003)

Publique um `pagamento.sucesso` com o publicador manual:

```bash
make publicar-pagamento RESERVA=9982a1b3-44c1-4221-a123-902183120192 \
                        USUARIO=c394c8b3-76a1-4328-b803-02f5923b7a15
```

**Esperado**: uma linha em `ingressos_emitidos` com `status='VALIDO'`, `codigo_qr`
começando em `CIN1.`, e uma linha em `registros_notificacao` com `status='ENVIADO'`.

```bash
psql "$DATABASE_URL" -c \
  "SELECT id, status, left(codigo_qr, 5) AS prefixo FROM notificacao.ingressos_emitidos;"
psql "$DATABASE_URL" -c \
  "SELECT ingresso_id, canal, status FROM notificacao.registros_notificacao;"
```

## Cenário 2 — Reentrega não duplica (FR-004, SC-001)

Publique **o mesmo** anúncio outra vez. Esperado: nenhum ingresso novo, `codigo_qr`
inalterado, e **nenhum** registro de aviso novo (D6 — a reentrega é inerte).

```bash
make publicar-pagamento RESERVA=9982a1b3-44c1-4221-a123-902183120192 \
                        USUARIO=c394c8b3-76a1-4328-b803-02f5923b7a15
psql "$DATABASE_URL" -c "SELECT count(*) FROM notificacao.ingressos_emitidos;"       # esperado: 1
psql "$DATABASE_URL" -c "SELECT count(*) FROM notificacao.registros_notificacao;"    # esperado: 1
```

## Cenário 3 — A pessoa vê os próprios ingressos (P3, FR-013, FR-023, FR-024)

```bash
TOKEN=$(./scripts/token-teste.sh c394c8b3-76a1-4328-b803-02f5923b7a15)

curl -s -H "Authorization: Bearer $TOKEN" \
  localhost:8080/api/v1/ingressos/meus-ingressos | jq

curl -s -H "Authorization: Bearer $TOKEN" \
  'localhost:8080/api/v1/ingressos/meus-ingressos?status=VALIDO' | jq

curl -si -H "Authorization: Bearer $TOKEN" \
  'localhost:8080/api/v1/ingressos/meus-ingressos?status=INVENTADO' | head -1   # 400

TOKEN_OUTRA=$(./scripts/token-teste.sh 00000000-0000-4000-8000-000000000000)
curl -s -H "Authorization: Bearer $TOKEN_OUTRA" \
  localhost:8080/api/v1/ingressos/meus-ingressos | jq   # esperado: []

curl -si localhost:8080/api/v1/ingressos/meus-ingressos | head -1                # 401
```

## Cenário 4 — A portaria valida, uma vez só (P2, FR-007, FR-008)

```bash
CODIGO=$(psql -tA "$DATABASE_URL" -c "SELECT codigo_qr FROM notificacao.ingressos_emitidos LIMIT 1;")

# primeira leitura: 200, autorizada
curl -si -X POST localhost:8080/api/v1/ingressos/validar \
  -H "X-API-Key: $PORTARIA_API_KEY" -H 'Content-Type: application/json' \
  -d "{\"codigo_qr\":\"$CODIGO\"}"

# segunda leitura: 409, já utilizado
curl -si -X POST localhost:8080/api/v1/ingressos/validar \
  -H "X-API-Key: $PORTARIA_API_KEY" -H 'Content-Type: application/json' \
  -d "{\"codigo_qr\":\"$CODIGO\"}" | head -1
```

Confirme que `utilizado_em` **não mudou** entre a primeira e a segunda leitura (FR-008).

## Cenário 5 — Falsificação e credencial errada (FR-010, FR-012)

As três respostas abaixo devem ser **idênticas**, corpo inclusive:

```bash
for C in "lixo" "CIN1.aaaa.bbbb" "${CODIGO%?}X"; do
  curl -s -X POST localhost:8080/api/v1/ingressos/validar \
    -H "X-API-Key: $PORTARIA_API_KEY" -H 'Content-Type: application/json' \
    -d "{\"codigo_qr\":\"$C\"}"; echo
done
```

Sem chave, e com chave errada — 401 nos dois casos, e o ingresso não é consultado:

```bash
curl -si -X POST localhost:8080/api/v1/ingressos/validar \
  -H 'Content-Type: application/json' -d '{"codigo_qr":"x"}' | head -1
curl -si -X POST localhost:8080/api/v1/ingressos/validar \
  -H 'X-API-Key: errada' -H 'Content-Type: application/json' \
  -d '{"codigo_qr":"x"}' | head -1
```

Credencial trocada de rota — 401 nas duas (edge case, D7):

```bash
curl -si -X POST localhost:8080/api/v1/ingressos/validar \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"codigo_qr\":\"$CODIGO\"}" | head -1
curl -si -H "X-API-Key: $PORTARIA_API_KEY" \
  localhost:8080/api/v1/ingressos/meus-ingressos | head -1
```

## Cenário 6 — Falha do aviso não derruba a emissão (P4, FR-018, FR-025)

```bash
export NOTIFICADOR_MODO=falhar    # o adaptador simulado passa a sempre falhar
make run
make publicar-pagamento RESERVA=$(uuidgen) USUARIO=c394c8b3-76a1-4328-b803-02f5923b7a15
```

**Esperado**: ingresso emitido e `VALIDO`; registro com `status='FALHA'` e `detalhes`
preenchido; **fila principal vazia** — a mensagem foi confirmada, não reprocessada.

```bash
psql "$DATABASE_URL" -c \
  "SELECT status, detalhes IS NOT NULL AS tem_detalhe FROM notificacao.registros_notificacao ORDER BY enviado_em DESC LIMIT 1;"
docker compose exec rabbitmq rabbitmqctl list_queues name messages
```

## Cenário 7 — Malformado vai para a quarentena, transitório volta (FR-022)

```bash
make publicar-cru PAYLOAD='{"reserva_id":"nao-e-uuid"}'
docker compose exec rabbitmq rabbitmqctl list_queues name messages
# esperado: notificacao.pagamento-sucesso.dlq com 1; principal com 0
```

Falha transitória — derrube o banco, publique um anúncio válido e observe a mensagem
voltando para a fila até esgotar `AMQP_LIMITE_ENTREGAS` (padrão 3) e cair na fila morta:

```bash
docker compose stop postgres
make publicar-pagamento RESERVA=$(uuidgen) USUARIO=c394c8b3-76a1-4328-b803-02f5923b7a15
sleep 5 && docker compose exec rabbitmq rabbitmqctl list_queues name messages
docker compose start postgres
```

## Cenário 8 — Log não vaza o código de acesso (FR-021, D13)

Com o serviço rodando, valide um ingresso e confirme que o código **não** aparece:

```bash
make run 2>&1 | tee /tmp/notificacao.log &
# ... execute o cenário 4 ...
grep -c "$CODIGO" /tmp/notificacao.log     # esperado: 0
grep -c "ingresso_id" /tmp/notificacao.log # esperado: > 0
```

---

## Bateria automatizada

```bash
make test              # domínio, casos de uso e contrato HTTP — sem docker
make test-integration  # Testcontainers: concorrência, quarentena, percurso completo, medições
make lint
```

Testes de integração cobrem o que os cenários manuais não conseguem afirmar com
confiança: duas entregas simultâneas da mesma reserva (SC-001), duas leituras
simultâneas do mesmo código (SC-004) e os três critérios de latência (SC-002, SC-003 e
SC-009), que exigem volume e medição de percentil.

---

## Mapa cenário → critério de sucesso

Os cenários deste roteiro **exercitam** comportamento; eles não medem tempo nem volume.
A distinção importa: um cenário que passa prova que o caminho existe, não que ele
responde dentro do prazo prometido. Os três critérios de latência são verificados por
teste automatizado de medição (T051 e T052), não aqui.

| Cenário | Verifica | Critério |
|---|---|---|
| 1, 2 | um ingresso por reserva, reentrega inerte | SC-001 |
| 3 | listagem recortada, ordenada e filtrada; sem acesso cruzado | SC-006 |
| 4 | primeira leitura autoriza, segunda nega | SC-004 (comportamento) |
| 5 | nenhuma falsificação aceita; credencial errada recusada | SC-005, SC-006 |
| 6 | falha de aviso registrada, entrada não impedida, mensagem confirmada | SC-007 |
| 7 | malformado em inspeção humana; transitório se recupera | SC-008, SC-010 |
| 8 | auditoria sem vazar o código de acesso | FR-021 |

| Não coberto aqui | Onde é medido | Medido em 2026-08-30 |
|---|---|---|
| SC-002 (ingresso disponível em < 5 s, p95) | `test/integration/vazao_test.go` | **53,6 ms** |
| SC-003 (veredito em < 1 s, p99, com 60 leituras simultâneas) | `test/integration/vazao_test.go` | **22,4 ms** |
| SC-009 (listagem em < 2 s, p95, com 200 ingressos) | `test/integration/vazao_listagem_test.go` | **0,54 ms** |

Os três estão uma a duas ordens de grandeza abaixo do alvo. A folga é grande o bastante
para que o teste continue verde sob máquina carregada, e pequena o bastante para que uma
regressão séria — um índice perdido, um N+1 introduzido — o derrube.
