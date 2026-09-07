# Servico-Catalogo

Ponto de entrada para clientes navegarem pelo catálogo de filmes, cinemas, salas
e sessões, e para iniciarem a reserva de poltronas. Expõe uma API REST pública,
atua como cliente gRPC do `Servico-Estoque` no momento da reserva — por um canal
mTLS — e anuncia no RabbitMQ a sessão criada, com a planta da sala.

Especificação, plano e tarefas: [`specs/001-catalogo-sessoes-reserva/`](specs/001-catalogo-sessoes-reserva/).
Princípios que governam o código: [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

Cinco dessas regras valem também para quem só vai ler ou mexer no código: a entrega
de um fato é ao menos uma vez, e nunca exatamente uma (VI); nada de complexidade
além da necessária ou pedida (VII); domínio e operações expostas têm teste
automatizado (VIII); **o código é a fonte da verdade, não a spec** (IX) — os
documentos acima são instrumentos de projeto, e afirmação sobre comportamento atual
se verifica no código; e divergência entre código e spec é pergunta ao mantenedor,
não decisão de quem encontrou (X).

## Duas divergências em relação à ERS original

Quem já leu `ers-catalogo.md` precisa saber destas duas mudanças deliberadas:

1. **Coleções respondem com envelope, não com array nu.** `GET /api/v1/filmes` e
   `GET /api/v1/sessoes` devolvem `{"itens": [...], "pagina": {...}}`. A paginação
   é obrigatória em toda consulta de coleção, e um array não comporta o total nem
   a indicação de próxima página.
2. **O 409 é `problem+json`, não `{"sucesso": false, "mensagem": "..."}`.** Todas
   as respostas de erro seguem a RFC 9457, com um `type` estável por categoria.
   Manter dois formatos de erro no mesmo serviço obrigaria o cliente a tratar
   cada rota de um jeito.

3. **Filmes aceitam escrita.** A ERS previa só a leitura do catálogo; hoje o
   recurso tem os quatro verbos (ver abaixo). O `DELETE` é lógico — o filme
   passa a `FORA_DE_CARTAZ` — porque as sessões gravadas referenciam o filme e
   apagá-lo romperia a grade.

O catálogo de erros está em [`specs/001-catalogo-sessoes-reserva/contracts/errors.md`](specs/001-catalogo-sessoes-reserva/contracts/errors.md),
e o contrato do fato publicado em [`contracts/eventos.md`](specs/001-catalogo-sessoes-reserva/contracts/eventos.md).

## O fato `sessao.criada`

Criar uma sessão publica `sessao.criada` no exchange `cinema.eventos`, com a
planta da sala expandida assento a assento — é dela que o `Servico-Estoque`
provisiona a matriz de poltronas daquela sessão. O catálogo declara só o
exchange: fila é de quem consome.

O fato é gravado numa caixa de saída (`catalogo.outbox_eventos`) **na mesma
transação** que insere a sessão, e um processo à parte a drena e republica até o
broker confirmar. Disso decorrem três coisas que quem integra precisa saber:

- a resposta do `POST` não espera pela publicação — uma sessão é criada com
  sucesso mesmo com o broker fora do ar, e o fato sai quando ele voltar;
- a entrega é **ao menos uma vez**: a mesma mensagem pode chegar repetida, e o
  consumidor descarta pelo `sessao_id`, que também vai no `message_id`;
- alterar ou cancelar uma sessão **não** emite fato — não há consumidor para
  isso, e o contrato registra a consequência.

O contexto de rastreamento da requisição viaja nos cabeçalhos da mensagem, de
modo que o span de quem consome não nasça órfão.

## O canal com o estoque

A reserva é a única chamada síncrona que este serviço faz, e ela vai ao
`Servico-Estoque` por gRPC sobre **mTLS**: o estoque exige certificado de
cliente, e é por ele que sabe quem está chamando — o `usuario_id` vai no corpo
justamente porque a identidade do serviço vem do certificado, não do payload.

O material é de desenvolvimento e sai de `make certs` no estoque, que emite um
par de cliente com `CN=servico-catalogo` assinado pela mesma CA do servidor. O
diretório `estoque/certs/` **não é versionado**, então gerar os certificados é
pré-requisito para subir o catálogo — não só o estoque.

Sem as três variáveis de certificado o processo recusa subir, como qualquer outra
configuração obrigatória. Não há modo em texto claro.

### Limite conhecido: erro de entrada chega como 503

O catálogo traduz **qualquer** erro do estoque em `503 estoque-indisponivel`. O
estoque, porém, distingue categorias no seu contrato de erros, e algumas delas
são culpa de quem chama:

| O que aconteceu | Devia responder | Responde hoje |
|---|---|---|
| Poltrona que não existe na sala | 409 ou 422 | 503 |
| Sessão sem matriz de poltronas provisionada | 409 ou 422 | 503 |
| Mais de 10 poltronas num bloqueio | 400 | 503 |
| Rótulo de poltrona fora do formato | 400 | 503 |

Isso ficou escondido enquanto o catálogo falava com um dublê que nunca devolvia
erro de gRPC. **Se você recebeu um 503 dizendo que o estoque está indisponível,
confira primeiro a entrada** — a mensagem culpa a infraestrutura, e o defeito
pode ser da requisição. Fechar isso é mapear as categorias de
`estoque/specs/001-estoque-bloqueio-poltronas/contracts/erros.md` para os status
certos, em `internal/adapter/estoque/mapper.go`.

O caminho feliz não é afetado: toda sessão criada por esta API tem matriz
provisionada, porque ela publica `sessao.criada`.

## O preço da reserva sai daqui

`POST /sessoes/{id}/reservar` não recebe preço: o corpo pede poltronas, e o
serviço calcula `valor_total` como o `preco_base` da sessão vezes o número de
poltronas — a mesma escolha feita com `capacidade_total`, e pelo mesmo motivo. O
valor segue ao estoque na solicitação de bloqueio e chega a quem cobra pelo fato
`reserva.criada`.

Este serviço é a autoridade do preço porque é o dono do cadastro da sessão.
Recalculá-lo em qualquer outro lugar duplicaria a regra.

## Superfície da API

| Método | Caminho | Credencial |
| --- | --- | --- |
| `GET` | `/api/v1/filmes` | pública |
| `GET` | `/api/v1/filmes/{id}` | pública |
| `POST` | `/api/v1/filmes` | Bearer |
| `PUT` | `/api/v1/filmes/{id}` | Bearer |
| `DELETE` | `/api/v1/filmes/{id}` | Bearer |
| `GET` | `/api/v1/cinemas` | pública |
| `GET` | `/api/v1/cinemas/{id}` | pública |
| `POST` | `/api/v1/cinemas` | Bearer |
| `PUT` | `/api/v1/cinemas/{id}` | Bearer |
| `DELETE` | `/api/v1/cinemas/{id}` | Bearer |
| `GET` | `/api/v1/salas` | pública |
| `GET` | `/api/v1/salas/{id}` | pública |
| `POST` | `/api/v1/salas` | Bearer |
| `PUT` | `/api/v1/salas/{id}` | Bearer |
| `DELETE` | `/api/v1/salas/{id}` | Bearer |
| `GET` | `/api/v1/sessoes` | pública |
| `POST` | `/api/v1/sessoes/{id}/reservar` | Bearer |
| `GET` | `/health` | pública |

Sobre a escrita de filmes: o identificador é gerado pelo serviço e volta no
corpo e no header `Location`; o `PUT` é substituição total, e campo opcional
omitido volta a ficar ausente; sem `status` no corpo o filme fica `EM_CARTAZ`;
e `GET /filmes/{id}` enxerga qualquer situação, inclusive `FORA_DE_CARTAZ` — o
recorte público vale só para a listagem.

A escrita de cinemas segue as mesmas regras, com uma diferença no `DELETE`: a
remoção marca o cinema como inativo em vez de apagar a linha, porque as salas o
referenciam e as sessões referenciam as salas. O cinema some de
`GET /cinemas`, segue legível em `GET /cinemas/{id}` e é alcançável pelo filtro
`GET /cinemas?ativo=false`.

As salas moram fora do caminho do cinema: cada uma é endereçada por
`/salas/{id}`, e o cinema é o filtro opcional `GET /salas?cinema_id=<uuid>` — sem
ele a listagem é da rede inteira, com ele o cinema precisa existir, ou a resposta
é `404`. O `cinema_id` vai no corpo da escrita e é do cadastro, não do estado que
o `PUT` redesenha: informar outro cinema responde `409`, e mudar a sala de cinema
não é uma operação da API. O `DELETE` também é lógico, pelo mesmo motivo do
cinema — as sessões referenciam a sala —, e o número liberado volta a ficar
disponível para a sala que a substituir.

A sala declara sua planta em `fileiras`: cada fileira tem uma letra, uma
quantidade de assentos e um tipo (`NORMAL`, `PCD` ou `NAMORADEIRA`, os mesmos que
o estoque aceita), e a fileira é uniforme — um assento PCD no meio de uma fileira
comum se declara como fileira própria. `capacidade_total` deixou de ser um número
digitado: ela é a soma dos assentos, calculada pelo serviço, e sai só na
resposta. Mandá-la no corpo da escrita responde `400`.

## Executando localmente

O compose da raiz do repositório sobe o catálogo com tudo de que ele depende —
Keycloak (com o realm `cinema` já importado de `keycloak/realm-cinema.json`), o
estoque simulado, o PostgreSQL e as migrações. Os quatro serviços dividem uma
instância e um banco (`cinema`), cada um dono do seu schema; os papéis e schemas
nascem em `../infra/postgres/init/`, aplicado no primeiro boot do volume. Os
dados ficam no volume `postgres-dados` e sobrevivem a `docker compose down` —
só `down -v` os apaga.

A porta publicada no host é a 5434, e não a 5432, que costuma estar ocupada pelo
PostgreSQL da própria máquina:

```bash
docker compose -f ../docker-compose.yml up -d catalogo
psql "postgres://catalogo:catalogo@localhost:5434/cinema?sslmode=disable" \
  -f test/fixtures/catalogo_exemplo.sql      # catálogo de exemplo, opcional
curl -s localhost:8082/health
```

A API fica em `http://localhost:8082`. As migrações rodam num serviço próprio
(`migrate`) que precisa terminar com sucesso antes de o catálogo subir, então o
serviço nunca encontra um esquema pela metade.

As tabelas do catálogo vivem no schema `catalogo`, não em `public`: as migrações
criam o schema e qualificam cada objeto, e o serviço fixa o `search_path` no
pool de conexões. Para inspecionar o banco com `psql`, aponte o `search_path`
antes (`SET search_path TO catalogo;`) ou qualifique as tabelas. A tabela de
controle do golang-migrate (`schema_migrations`) mora no mesmo schema: em
`public` os quatro serviços disputariam uma só.

### Documentação da API

| Recurso | Endereço |
| --- | --- |
| Swagger UI | `http://localhost:8082/docs` |
| Contrato OpenAPI 3.1 | `http://localhost:8082/openapi.yaml` |

O contrato não é gerado a partir do código: ele é escrito à mão em
`specs/001-catalogo-sessoes-reserva/contracts/openapi.yaml`, embutido no binário
com `go:embed` e servido como está. Depois de editá-lo, rode `make openapi-sync`
para atualizar a cópia de runtime — `make test` falha se as duas divergirem, e
falha também se o contrato descrever uma rota que o roteador não registra (ou o
contrário).

As rotas protegidas (a escrita de filmes e `POST /sessoes/{id}/reservar`)
aceitam duas credenciais, ambas validadas do mesmo jeito: assinatura RS256
conferida contra o JWKS do realm, mais `iss`, `aud` e expiração.

**Usuário humano** (`teste`/`teste`, client público `cinema-app`). Gere o token
e cole-o em **Authorize** — apenas o valor, sem o prefixo `Bearer`:

```bash
TOKEN=$(curl -s -d client_id=cinema-app -d username=teste -d password=teste \
  -d grant_type=password \
  http://localhost:8081/realms/cinema/protocol/openid-connect/token | jq -r .access_token)
```

**Máquina a máquina** (client confidencial `cinema-m2m`, com service account).
Não há usuário no meio: o `sub` do token é o da service account.

```bash
TOKEN=$(curl -s -d client_id=cinema-m2m -d client_secret=segredo-de-desenvolvimento \
  -d grant_type=client_credentials \
  http://localhost:8081/realms/cinema/protocol/openid-connect/token | jq -r .access_token)
```

No Swagger, esse fluxo aparece em **Authorize** como `clientCredentials`, com
campos de client_id e client_secret. **Use-o só em desenvolvimento:** quem chama
o token endpoint ali é o navegador, então o secret trafega pelo browser e o
client deixa de ser confidencial na prática — em produção o secret pertence ao
cofre de quem chama a API, nunca à página. O `cinema-m2m` do
`keycloak/realm-cinema.json` existe para isso: secret fixo, de desenvolvimento.

Os dois clients trazem o mesmo `oidc-audience-mapper`, que injeta
`aud: cinema-app` no access token. Ele não é decoração: sem o mapper o Keycloak
emite `aud: account` no fluxo client_credentials e a API recusa o token com 401
— é o tropeço mais comum ao ligar M2M.

O realm fixa o emissor em `http://keycloak:8081` — a mesma URL dentro e fora da
rede do compose. O `iss` do token precisa bater exatamente com o emissor que o
catálogo descobriu no boot, e um hostname por ambiente quebraria essa
verificação. O console de admin continua em `http://localhost:8081` (admin/admin).

### Rodando o serviço fora do contêiner

```bash
(cd ../estoque && make certs)   # material de desenvolvimento; não é versionado
docker compose -f ../docker-compose.yml up -d keycloak estoque rabbitmq migrate-catalogo
echo "127.0.0.1 keycloak" | sudo tee -a /etc/hosts   # uma vez, pelo emissor fixo
export DATABASE_URL="postgres://catalogo:catalogo@localhost:5434/cinema?sslmode=disable"   # precisa da query string: `make migrate-up` anexa `&search_path=catalogo`
export KEYCLOAK_ISSUER_URL="http://keycloak:8081/realms/cinema"
export KEYCLOAK_AUDIENCE="cinema-app"
export ESTOQUE_GRPC_ADDR="localhost:50051"
export ESTOQUE_TLS_CA_FILE="../estoque/certs/ca.pem"
export ESTOQUE_TLS_CERT_FILE="../estoque/certs/cliente.pem"
export ESTOQUE_TLS_KEY_FILE="../estoque/certs/cliente-key.pem"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
export HTTP_PORT=8082
make run
```

O roteiro completo de validação está em
[`specs/001-catalogo-sessoes-reserva/quickstart.md`](specs/001-catalogo-sessoes-reserva/quickstart.md).

### Como contêiner avulso

A imagem sobe apenas com variáveis de ambiente, sem arquivo de configuração:

```bash
docker build -t servico-catalogo .
docker run --rm -p 8080:8080 \
  -v "$PWD/../estoque/certs:/certs:ro" \
  -e DATABASE_URL="..." -e KEYCLOAK_ISSUER_URL="..." \
  -e KEYCLOAK_AUDIENCE="cinema-app" -e ESTOQUE_GRPC_ADDR="estoque:50051" \
  -e ESTOQUE_TLS_CA_FILE="/certs/ca.pem" \
  -e ESTOQUE_TLS_CERT_FILE="/certs/cliente.pem" \
  -e ESTOQUE_TLS_KEY_FILE="/certs/cliente-key.pem" \
  -e RABBITMQ_URL="amqp://guest:guest@rabbitmq:5672/" \
  servico-catalogo
curl -s localhost:8080/health
```

## Variáveis de ambiente

Obrigatórias — o processo **recusa subir** se qualquer uma faltar ou estiver
malformada, e a mensagem lista todas as pendências de uma vez:

| Variável | Descrição |
|---|---|
| `DATABASE_URL` | Conexão com o PostgreSQL (o `search_path` é fixado no código, no schema `catalogo`) |
| `KEYCLOAK_ISSUER_URL` | Emissor OIDC das credenciais |
| `KEYCLOAK_AUDIENCE` | Audiência esperada no token |
| `ESTOQUE_GRPC_ADDR` | Endereço gRPC do `Servico-Estoque` |
| `ESTOQUE_TLS_CA_FILE` | CA que assina o certificado do estoque |
| `ESTOQUE_TLS_CERT_FILE` | Certificado de cliente apresentado ao estoque |
| `ESTOQUE_TLS_KEY_FILE` | Chave do certificado de cliente |
| `RABBITMQ_URL` | Broker onde o fato `sessao.criada` é publicado |

Opcionais, com padrão:

| Variável | Padrão | Descrição |
|---|---|---|
| `HTTP_PORT` | `8080` | Porta do servidor |
| `ESTOQUE_TIMEOUT` | `2s` | Espera máxima pelo estoque; valores acima de 2s são recusados |
| `BREAKER_FALHAS_CONSECUTIVAS` | `5` | Falhas seguidas até entrar em recusa rápida |
| `BREAKER_INTERVALO_ABERTO` | `30s` | Tempo em recusa rápida antes de tentar de novo |
| `PAGINACAO_TAMANHO_PADRAO` | `20` | Tamanho de página quando não informado |
| `PAGINACAO_TAMANHO_MAXIMO` | `100` | Teto de `page_size`; acima disso a requisição é recusada |
| `OUTBOX_INTERVALO` | `1s` | De quanto em quanto tempo a caixa de saída é drenada |
| `OUTBOX_LOTE` | `100` | Fatos lidos por drenagem |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Coletor de rastros e métricas; sem ele o serviço roda e apenas não exporta |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |

## Arquitetura

Hexagonal, com a dependência apontando para dentro:

```
cmd/            composição — o único lugar onde o núcleo encontra a infraestrutura
internal/
  domain/       entidades e regras; não importa nada de infraestrutura
  usecase/      casos de uso e as portas que eles exigem
  adapter/      http (entrada), postgres, estoque, amqp, identidade (saída)
  platform/     configuração, observabilidade, saúde
```

A regra não depende de revisão humana: o `depguard` configurado em
`.golangci.yml` falha o build se `domain` ou `usecase` importarem um adaptador,
um driver, o framework web ou o provedor de identidade.

## Testes

```bash
make test              # unitários e de contrato, sem infraestrutura
make test-integration  # PostgreSQL real via Testcontainers (requer Docker)
make lint
```

Os testes de integração sobem um PostgreSQL em contêiner e um `Servico-Estoque`
simulado em memória. Provam, entre outras coisas: que uma recusa local nunca
contata o estoque; que 50 solicitações paralelas pelas mesmas poltronas resultam
em exatamente uma confirmação; que o contexto de rastreamento recebido chega ao
estoque; que as listagens usam os índices esperados; e que o anúncio da sessão e a
própria sessão são gravados de forma indivisível — uma sessão recusada não deixa
fato na caixa, e um fato cuja publicação falha volta na drenagem seguinte.

## Limite conhecido

A paginação é por deslocamento, e o total exato exige contar as linhas que
atendem ao filtro. Ambos crescem com o acervo. Nos volumes previstos o custo
medido é de dezenas de milissegundos, com folga larga sobre o orçamento — mas
acima de ~100.000 registros em uma coleção a saída é migrar para paginação por
cursor, o que muda o contrato e exige uma nova versão da API.
