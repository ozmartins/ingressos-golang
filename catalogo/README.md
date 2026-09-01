# Servico-Catalogo

Ponto de entrada para clientes navegarem pelo catálogo de filmes, cinemas, salas
e sessões, e para iniciarem a reserva de poltronas. Expõe uma API REST pública e
atua como cliente gRPC do `Servico-Estoque` no momento da reserva.

Especificação, plano e tarefas: [`specs/001-catalogo-sessoes-reserva/`](specs/001-catalogo-sessoes-reserva/).
Princípios que governam o código: [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

Quatro dessas regras valem também para quem só vai ler ou mexer no código: nada de
complexidade além da necessária ou pedida (VI); domínio e operações expostas têm
teste automatizado (VII); **o código é a fonte da verdade, não a spec** (VIII) — os
documentos acima são instrumentos de projeto, e afirmação sobre comportamento atual
se verifica no código; e divergência entre código e spec é pergunta ao mantenedor,
não decisão de quem encontrou (IX).

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

O catálogo de erros está em [`specs/001-catalogo-sessoes-reserva/contracts/errors.md`](specs/001-catalogo-sessoes-reserva/contracts/errors.md).

## Executando localmente

O compose da raiz do repositório sobe o catálogo com tudo de que ele depende —
Keycloak (com o realm `cinema` já importado de `keycloak/realm-cinema.json`), o
estoque simulado e as migrações. O PostgreSQL não sobe em contêiner: os serviços
usam a instância instalada na máquina, preparada uma única vez com
`psql -U postgres -f ../infra/postgres/criar-bancos.sql`:

```bash
docker compose -f ../docker-compose.yml up -d catalogo
psql "postgres://catalogo:catalogo@localhost:5432/catalogo?sslmode=disable" \
  -f test/fixtures/catalogo_exemplo.sql      # catálogo de exemplo, opcional
curl -s localhost:8082/health
```

A API fica em `http://localhost:8082`. As migrações rodam num serviço próprio
(`migrate`) que precisa terminar com sucesso antes de o catálogo subir, então o
serviço nunca encontra um esquema pela metade.

As tabelas do catálogo vivem no schema `catalogo`, não em `public`: as migrações
criam o schema e qualificam cada objeto, e o serviço fixa o `search_path` no
pool de conexões. Para inspecionar o banco com `psql`, aponte o `search_path`
antes (`SET search_path TO catalogo;`) ou qualifique as tabelas. Só a tabela de
controle do golang-migrate (`schema_migrations`) segue em `public` — ela é do
ferramental, não do domínio.

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

Para exercitar `POST /sessoes/{id}/reservar` pela interface, gere o token abaixo
e cole-o em **Authorize** (apenas o valor, sem o prefixo `Bearer`).

Token para as rotas autenticadas (usuário de desenvolvimento `teste`/`teste`):

```bash
TOKEN=$(curl -s -d client_id=cinema-app -d username=teste -d password=teste \
  -d grant_type=password \
  http://localhost:8081/realms/cinema/protocol/openid-connect/token | jq -r .access_token)
```

O realm fixa o emissor em `http://keycloak:8081` — a mesma URL dentro e fora da
rede do compose. O `iss` do token precisa bater exatamente com o emissor que o
catálogo descobriu no boot, e um hostname por ambiente quebraria essa
verificação. O console de admin continua em `http://localhost:8081` (admin/admin).

### Rodando o serviço fora do contêiner

```bash
docker compose -f ../docker-compose.yml up -d keycloak estoque-simulado migrate-catalogo
echo "127.0.0.1 keycloak" | sudo tee -a /etc/hosts   # uma vez, pelo emissor fixo
export DATABASE_URL="postgres://catalogo:catalogo@localhost:5432/catalogo?sslmode=disable"   # precisa da query string: `make migrate-up` anexa `&search_path=public`
export KEYCLOAK_ISSUER_URL="http://keycloak:8081/realms/cinema"
export KEYCLOAK_AUDIENCE="cinema-app"
export ESTOQUE_GRPC_ADDR="localhost:50052"   # o simulado; a 50051 é do estoque real
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
  -e DATABASE_URL="..." -e KEYCLOAK_ISSUER_URL="..." \
  -e KEYCLOAK_AUDIENCE="cinema-app" -e ESTOQUE_GRPC_ADDR="..." \
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

Opcionais, com padrão:

| Variável | Padrão | Descrição |
|---|---|---|
| `HTTP_PORT` | `8080` | Porta do servidor |
| `ESTOQUE_TIMEOUT` | `2s` | Espera máxima pelo estoque; valores acima de 2s são recusados |
| `BREAKER_FALHAS_CONSECUTIVAS` | `5` | Falhas seguidas até entrar em recusa rápida |
| `BREAKER_INTERVALO_ABERTO` | `30s` | Tempo em recusa rápida antes de tentar de novo |
| `PAGINACAO_TAMANHO_PADRAO` | `20` | Tamanho de página quando não informado |
| `PAGINACAO_TAMANHO_MAXIMO` | `100` | Teto de `page_size`; acima disso a requisição é recusada |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Coletor de rastros e métricas; sem ele o serviço roda e apenas não exporta |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |

## Arquitetura

Hexagonal, com a dependência apontando para dentro:

```
cmd/            composição — o único lugar onde o núcleo encontra a infraestrutura
internal/
  domain/       entidades e regras; não importa nada de infraestrutura
  usecase/      casos de uso e as portas que eles exigem
  adapter/      http (entrada), postgres, estoque, identidade (saída)
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
estoque; e que as listagens usam os índices esperados.

## Limite conhecido

A paginação é por deslocamento, e o total exato exige contar as linhas que
atendem ao filtro. Ambos crescem com o acervo. Nos volumes previstos o custo
medido é de dezenas de milissegundos, com folga larga sobre o orçamento — mas
acima de ~100.000 registros em uma coleção a saída é migrar para paginação por
cursor, o que muda o contrato e exige uma nova versão da API.
