# Servico-Catalogo

Ponto de entrada para clientes navegarem pelo catálogo de filmes, cinemas, salas
e sessões, e para iniciarem a reserva de poltronas. Expõe uma API REST pública e
atua como cliente gRPC do `Servico-Estoque` no momento da reserva.

Especificação, plano e tarefas: [`specs/001-catalogo-sessoes-reserva/`](specs/001-catalogo-sessoes-reserva/).
Princípios que governam o código: [`.specify/memory/constitution.md`](.specify/memory/constitution.md).

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

```bash
docker compose up -d                  # postgres, keycloak, estoque simulado
export DATABASE_URL="postgres://catalogo:catalogo@localhost:5432/catalogo?sslmode=disable"
migrate -path migrations -database "$DATABASE_URL" up
psql "$DATABASE_URL" -f test/fixtures/catalogo_exemplo.sql

export KEYCLOAK_ISSUER_URL="http://localhost:8081/realms/cinema"
export KEYCLOAK_AUDIENCE="cinema-app"
export ESTOQUE_GRPC_ADDR="localhost:50051"
make run
```

O roteiro completo de validação está em
[`specs/001-catalogo-sessoes-reserva/quickstart.md`](specs/001-catalogo-sessoes-reserva/quickstart.md).

### Como contêiner

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
| `DATABASE_URL` | Conexão com o PostgreSQL |
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
