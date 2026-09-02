# Servico-Notificacao

Emite os ingressos digitais da plataforma de cinema a partir de pagamentos
confirmados, avisa a pessoa e disponibiliza o bilhete para o aplicativo do
cliente e para a portaria da sala.

É a ponta da cadeia: consome `pagamento.sucesso` do RabbitMQ e **não publica
evento nenhum**.

## O que faz

| Operação | Como | Quem usa |
|---|---|---|
| Emitir ingresso | consumo de `pagamento.sucesso` | assíncrono, ninguém espera |
| Listar meus ingressos | `GET /api/v1/ingressos/meus-ingressos` | aplicativo do cliente (JWT) |
| Validar na entrada | `POST /api/v1/ingressos/validar` | catraca da portaria (`X-API-Key`) |

Contrato completo em
[`specs/001-emissao-ingressos/contracts/openapi.yaml`](specs/001-emissao-ingressos/contracts/openapi.yaml).

## Três decisões que explicam o código

1. **A emissão única mora no banco.** `UNIQUE (reserva_id)` mais
   `INSERT ... ON CONFLICT DO NOTHING RETURNING *`: a linha do ingresso **é** o
   registro de "já processei". Não há tabela de mensagens processadas nem trava
   distribuída.
2. **A baixa na portaria é uma escrita condicionada.**
   `UPDATE ... WHERE id = $1 AND status = 'VALIDO'`. Uma linha afetada é a
   autorização; o motivo da recusa é uma segunda pergunta, feita só quando a
   primeira já disse não. É o que garante uma única autorização sob leituras
   simultâneas.
3. **A falha do aviso morre onde acontece.** O erro do notificador é capturado e
   nunca propagado: o ingresso continua válido e a mensagem é confirmada. Uma
   falha de e-mail não pode virar reprocessamento.

O raciocínio completo, com as alternativas rejeitadas, está em
[`research.md`](specs/001-emissao-ingressos/research.md).

## Variáveis de ambiente

Variável ausente ou malformada **impede o processo de subir**, e o erro lista
todas as pendências de uma vez. Valor inválido numa variável com padrão também é
erro de largada — não cai silenciosamente no padrão.

**Obrigatórias, sem padrão**

| Variável | Para quê |
|---|---|
| `DATABASE_URL` | PostgreSQL (o `search_path` é fixado no código, no schema `notificacao`) |
| `AMQP_URL` | RabbitMQ |
| `JWKS_URL`, `JWT_ISSUER`, `JWT_AUDIENCE` | validação do token do Keycloak |
| `INGRESSO_QR_SEGREDO` | assinatura do código de acesso |
| `PORTARIA_API_KEY` | credencial dos dispositivos de portaria |

> Subir sem `INGRESSO_QR_SEGREDO` emitiria ingressos que a portaria nunca
> validaria. Por isso o processo recusa subir, e por isso o segredo **nunca** é
> gerado automaticamente: cada reinício invalidaria todos os ingressos já
> emitidos.

**Com padrão**

| Variável | Padrão |
|---|---|
| `PORTA_HTTP` | `8080` |
| `AMQP_EXCHANGE` | `cinema.eventos` |
| `AMQP_EXCHANGE_DLX` | `cinema.eventos.dlx` |
| `AMQP_FILA_PAGAMENTO_SUCESSO` | `notificacao.pagamento-sucesso` |
| `AMQP_PREFETCH` | `10` |
| `AMQP_LIMITE_ENTREGAS` | `3` (tentativas, não reentregas) |
| `NOTIFICADOR_MODO` | `enviar` (`falhar` exercita o caminho de erro) |
| `NIVEL_LOG` | `info` |

## Rodar

```bash
docker compose -f ../docker-compose.yml up -d rabbitmq keycloak
export DATABASE_URL='postgres://notificacao:notificacao@localhost:5434/cinema?sslmode=disable'
make migrate-up
make run
```

As tabelas da notificação vivem no schema `notificacao`, não em `public`: a
migração cria o schema e qualifica cada objeto, e o serviço fixa o `search_path`
no pool de conexões. Para inspecionar o banco com `psql`, aponte o `search_path`
antes (`SET search_path TO notificacao;`) ou qualifique as tabelas. A tabela
de controle do golang-migrate (`schema_migrations`) mora no mesmo schema: em
`public` os quatro serviços disputariam uma só.

## Documentação da API

O contrato é servido pelo próprio serviço: `/openapi.yaml` devolve o documento e
`/docs` abre o Swagger UI sobre ele — http://localhost:8084/docs no compose. Nenhum dos dois exige credencial:
pedir token para ler o contrato barraria justamente quem ainda vai integrar.

O contrato versionado em `specs/001-emissao-ingressos/contracts/openapi.yaml` é a fonte da verdade; a cópia embutida no
binário é gerada por `make openapi-sync`, e um teste de paridade falha quando as
duas divergem.

Roteiro completo de validação ponta a ponta, com oito cenários, em
[`quickstart.md`](specs/001-emissao-ingressos/quickstart.md).

## Testar

```bash
make test              # domínio, casos de uso e contrato HTTP — sem docker
make test-integration  # Testcontainers: concorrência, quarentena, medições
make lint
```

Domínio e operações expostas têm teste obrigatório, por exigência do princípio II
da constituição do workspace. Percentual de linhas cobertas não é portão em
lugar nenhum deste serviço.

## Uma regra que o código guarda em silêncio

O `codigo_qr` **nunca** vai para log, atributo de rastro ou mensagem de erro. Log
é copiado, agregado e lido por muita gente — um código de acesso em log é um
ingresso utilizável em log, e a assinatura não protege contra quem simplesmente
copia o código inteiro. O que identifica a operação nos registros é o
`ingresso_id`. Há teste que cobra isso.
