# Phase 0 — Research: Catálogo de Filmes, Sessões e Reserva de Poltronas

**Feature**: `001-catalogo-sessoes-reserva` | **Date**: 2026-08-29

A ERS (`ers-catalogo.md`) já fixou linguagem, arquitetura, protocolos, banco e provedor de identidade. Este documento não os reabre — registra as decisões que ainda estavam em aberto abaixo dessas escolhas, e as duas tensões reais entre requisitos que o desenho precisou resolver.

## Unknowns herdados do Technical Context

Nenhum marcador NEEDS CLARIFICATION restou. Cada lacuna abaixo foi decidida com base na ERS, na spec e nos volumes fixados por SC-003.

---

## D1 — Roteamento HTTP

**Decision**: `net/http.ServeMux` da biblioteca padrão, com padrões de método e caminho (`GET /api/v1/filmes`, `GET /api/v1/cinemas/{id}/salas`).

**Rationale**: a ERS exige Go 1.22+, versão em que o mux da stdlib passou a suportar método e variáveis de caminho — exatamente o que os 6 endpoints desta feature precisam. Um roteador externo acrescentaria dependência, superfície de atualização e vocabulário próprio sem entregar nada que o serviço use. A arquitetura hexagonal já isola o roteamento em um adaptador, então trocá-lo depois é barato.

**Alternatives considered**: `chi` (middlewares componíveis e sub-roteadores — desnecessário para 6 rotas planas); `gin` (rápido e popular, mas impõe seu próprio contexto e tratamento de erro, o que atrita com handlers finos que só traduzem para casos de uso); `echo` (mesmas objeções do gin).

---

## D2 — Acesso a dados

**Decision**: `pgx/v5` com pool de conexões e SQL escrito à mão dentro dos repositórios em `internal/adapter/postgres`.

**Rationale**: as consultas desta feature são poucas, fixas e com junções explícitas (sessão → sala → cinema → filme). SQL à mão deixa o plano de execução visível e sob controle, o que importa diretamente para SC-003. `pgx` é o driver nativo mais maduro do ecossistema e expõe tipos do PostgreSQL (`timestamptz`, `numeric`) sem conversões surpresa — relevante porque `preco_base` é `DECIMAL(10,2)` e não pode virar `float64` silenciosamente.

**Alternatives considered**: GORM (ORM completo; esconde o SQL justamente onde precisamos vê-lo, e carrega comportamento implícito de carregamento associado); `sqlc` (gera código tipado a partir do SQL — boa opção, descartada por adicionar etapa de geração ao build para um conjunto pequeno e estável de consultas); `database/sql` puro (perderia o mapeamento de tipos do pgx sem ganhar nada).

**Consequência**: `preco_base` trafega como `pgtype.Numeric` no adaptador e é convertido para uma representação decimal exata no domínio; nunca para ponto flutuante binário.

---

## D3 — Estratégia de paginação

**Decision**: paginação por deslocamento (`page` + `page_size`), com ordenação determinística `ORDER BY <chave natural>, id`, e o total obtido em uma **segunda consulta** (`SELECT COUNT(*)` com o mesmo filtro).

> **Revisado durante a implementação.** A decisão original trazia o total na mesma consulta, via `COUNT(*) OVER ()`. `EXPLAIN ANALYZE` mostrou que a janela obriga o planejador a materializar todo o conjunto filtrado antes de ordenar, o que descarta o índice de ordenação e transforma toda listagem em varredura completa — exatamente o que os índices existiam para evitar. Duas consultas indexadas custam menos que uma não indexada. Consequência aceita: as duas não compartilham transação, então uma escrita entre elas pode fazer o total divergir da página em um registro; para navegação de catálogo isso é irrelevante.

**Rationale**: FR-003 exige o total de registros na resposta, e FR-004 exige ordenação estável — o desempate por `id` garante que páginas consecutivas não repitam nem omitam registros. Deslocamento é a única forma que entrega o total e o acesso a uma página arbitrária, ambos exigidos pelo contrato. A contagem separada é o preço de manter a página indexada.

**Tensão registrada**: SC-003 diz que o tempo de resposta "não cresce com o total de registros armazenados", e paginação por deslocamento *degrada* em páginas profundas, porque o banco varre e descarta as linhas puladas. Nos volumes que a própria spec fixa (5.000 sessões, ~250 páginas de 20) essa degradação é irrelevante — a varredura descartada é de milhares de linhas indexadas, não milhões. A decisão é consciente e tem um limite conhecido.

Além do deslocamento, a contagem exata exigida por FR-003 tem o mesmo comportamento: `COUNT(*) OVER ()` percorre todas as linhas que atendem ao filtro. Nos volumes fixados isso é irrelevante; é o segundo motivo pelo qual o limite abaixo existe.

**Limite documentado**: acima de ~100.000 registros em uma coleção, ou se surgir requisito de varredura completa pelo cliente, migrar para paginação por chave (keyset/cursor). Isso mudaria o contrato — `page` daria lugar a um cursor opaco — e por isso deve ser tratado como versão nova da API, não como ajuste interno.

**Alternatives considered**: keyset desde já (não entrega o total exigido por FR-003 sem uma segunda consulta cara, e não permite pular para uma página arbitrária); deslocamento sem total (viola FR-003).

**Parâmetros**: `page` começa em 1; `page_size` padrão 20, máximo 100 (configurável por `PAGINACAO_TAMANHO_MAXIMO`). Página além do fim devolve lista vazia com o total correto (FR-005).

---

## D4 — Validação de credenciais

**Decision**: `github.com/coreos/go-oidc/v3/oidc` apontado para o issuer do Keycloak, usando `oidc.NewRemoteKeySet` para buscar e manter em cache as chaves públicas (JWKS), com verificação local de assinatura RS256, `exp`, `iss` e `aud`.

**Rationale**: FR-020 exige validação autônoma, sem consultar o emissor a cada requisição. O `RemoteKeySet` busca o JWKS uma vez, mantém em memória e só volta ao Keycloak quando encontra um `kid` desconhecido — o que também cobre rotação de chaves sem reinício. A biblioteca é a implementação de referência para OIDC em Go e trata os detalhes de verificação que são fáceis de errar à mão.

**Ponto de atenção**: o cache de JWKS é a *única* forma de cache no serviço, e não conflita com FR-010 — chave pública não é dado de catálogo, e servir uma chave em cache não expõe estado desatualizado do domínio.

**Alternatives considered**: `lestrrat-go/jwx/v2` com `jwk.Cache` (equivalente em capacidade, com controle mais fino de atualização; descartado por exigir montar a verificação de claims manualmente); `golang-jwt/jwt` com JWKS buscado à mão (mais código nosso em terreno sensível a erro).

**Identidade**: `usuario_id` sai da claim `sub`. Token válido em assinatura mas sem `sub` é recusado (FR-021, edge case correspondente na spec).

---

## D5 — Cliente gRPC e política de falha

**Decision**: `grpc-go` com conexão única e duradoura ao `Servico-Estoque`; `context.WithTimeout(2s)` por chamada; **nenhuma** política de retentativa configurada; disjuntor `sony/gobreaker/v2` envolvendo a chamada.

**Rationale**: FR-027 fixa o teto de 2s e FR-029 proíbe repetir uma solicitação já enviada — retentar poderia criar um bloqueio órfão no estoque enquanto o cliente já recebeu erro. O disjuntor atende FR-030 e SC-007: após uma sequência de falhas consecutivas ele abre e passa a recusar em microssegundos, e o estado semiaberto testa a recuperação sozinho.

**Parâmetros iniciais** (todos configuráveis): abre após 5 falhas consecutivas; permanece aberto por 30 segundos; no estado semiaberto permite 1 chamada de prova. Isso satisfaz SC-007 com folga — retomada em até 30s contra o limite de 1 minuto.

**Mapeamento de desfechos**: `codes.OK` com `sucesso=true` → 201; `sucesso=false` → 409; `codes.DeadlineExceeded`, `codes.Unavailable` e disjuntor aberto → 503; resposta com `sucesso=true` mas sem `reserva_id` ou sem `expira_em` → 502, porque alegar sucesso sem esses dados quebraria o contrato com o cliente (edge case explícito na spec).

**Alternatives considered**: retentativa nativa do gRPC via service config (proibida por FR-029); disjuntor caseiro (a máquina de estados tem armadilhas de concorrência que uma biblioteca madura já resolveu); nova conexão por chamada (custo de handshake em cada reserva, sem benefício).

---

## D6 — Observabilidade

**Decision**: OpenTelemetry para rastreamento e métricas, exportador OTLP; `log/slog` com `JSONHandler` para logs; instrumentação `otelhttp` na entrada e `otelgrpc` na saída; `trace_id` e `span_id` injetados em todo registro de log.

**Rationale**: FR-036 exige aceitar o contexto de rastreamento do cliente e propagá-lo até o estoque — `otelhttp` extrai o cabeçalho `traceparent` (W3C) e `otelgrpc` o injeta nos metadados da chamada, o que entrega a correlação ponta a ponta de SC-011 sem código manual. `slog` é stdlib desde o Go 1.21 e dispensa dependência de logging.

**Log de auditoria da reserva** (FR-033): além do log de requisição comum a todas as rotas, cada solicitação de reserva emite um registro dedicado com `usuario_id`, `sessao_id`, as poltronas pedidas, o desfecho e o `reserva_id` quando houver. O token **nunca** é registrado — só o `sub` já extraído. Esse registro é o que permite responder "quem pediu o quê e o que aconteceu" sem depender do estoque.

**Métricas nomeadas** (FR-035): `http.server.request.duration` (por rota e status), `estoque.bloqueio.duration`, `estoque.bloqueio.total` rotulada por desfecho (`sucesso`, `indisponivel`, `timeout`, `recusa_rapida`) e `estoque.breaker.state`.

**Alternatives considered**: Prometheus client direto (bom para métricas, não resolve rastreamento; OTel exporta para Prometheus de qualquer forma); Zap ou Zerolog (mais rápidos que `slog`, sem diferença perceptível neste volume e com uma dependência a mais); rastreamento manual por cabeçalho de correlação próprio (reinventaria o W3C Trace Context e não interoperaria com o estoque).

---

## D7 — Formato de erro

**Decision**: RFC 9457 (`application/problem+json`) em toda resposta de erro, com um `type` estável por categoria.

**Rationale**: FR-038 e SC-009 exigem que o cliente distinga categorias sem interpretar texto livre. O `type` é um URI estável, versionável e documentado — o `title` e o `detail` podem mudar de redação sem quebrar cliente algum. É um padrão publicado, o que evita inventar um envelope próprio.

**Tensão registrada**: a ERS especifica, para o 409, um corpo `{"sucesso": false, "mensagem": "..."}`. O plano diverge deliberadamente: o corpo de conflito passa a ser um `problem+json` com `type: .../poltronas-indisponiveis`. Manter dois formatos de erro no mesmo serviço — um para 409, outro para o resto — anularia SC-009. **Esta divergência precisa ser comunicada a quem consome a ERS.**

**Alternatives considered**: envelope `{sucesso, mensagem}` uniforme conforme a ERS (não carrega categoria legível por máquina, e `sucesso: false` em uma resposta que já tem status HTTP de erro é redundante); erros só pelo status HTTP (insuficiente — 400 cobre muitos motivos distintos).

---

## D8 — Divergência de contrato: envelope de paginação

**Decision**: as coleções passam a responder com um objeto envelope (`{"itens": [...], "pagina": {...}}`) em vez do array nu que a ERS mostra.

**Rationale**: FR-003 exige total e indicação de mais páginas na resposta, o que um array nu não comporta. A mudança é consequência direta da decisão do usuário de tornar a paginação obrigatória em todas as consultas.

**Impacto**: `GET /api/v1/filmes` e `GET /api/v1/sessoes` mudam de forma em relação ao documento original. Como nenhum cliente foi construído ainda, o custo é zero agora e seria alto depois. **Também precisa ser comunicada a quem consome a ERS.**

---

## D9 — Estratégia de testes

**Decision**: três camadas — unitários de domínio e casos de uso com portas falsas; testes de contrato HTTP com `httptest` validando as respostas contra `contracts/openapi.yaml`; integração com PostgreSQL real via Testcontainers e um `Servico-Estoque` simulado servido em `grpc/test/bufconn`.

**Rationale**: os casos de uso são onde vivem as regras (situação da sessão, validação de poltronas, mapeamento de desfechos) e precisam rodar em milissegundos, sem infraestrutura. As consultas SQL, por outro lado, só provam algo contra um PostgreSQL de verdade — SQLite ou mocks de banco esconderiam justamente os erros de junção, ordenação e tipo decimal que importam aqui. O `bufconn` permite testar o cliente gRPC, o timeout e o disjuntor de forma determinística, incluindo um servidor que atrasa de propósito para provar SC-004 e SC-007.

**Alternatives considered**: mocks de `database/sql` (provam que o código chama o driver, não que a consulta está correta); banco compartilhado de testes (estado entre execuções, paralelismo frágil); testes de ponta a ponta com Keycloak real (lento; a verificação de token é testada com um emissor de JWKS local controlado pelo teste).

---

## D10 — Configuração

**Decision**: variáveis de ambiente lidas e validadas uma única vez na inicialização, em `internal/platform/config`; o processo falha ao subir se alguma obrigatória estiver ausente ou malformada.

**Rationale**: FR-032 e SC-010. Falhar cedo e ruidosamente é preferível a descobrir um endereço vazio na primeira reserva do dia.

**Variáveis**: `DATABASE_URL`, `KEYCLOAK_ISSUER_URL`, `KEYCLOAK_AUDIENCE`, `ESTOQUE_GRPC_ADDR`, `ESTOQUE_TIMEOUT` (padrão `2s`, teto 2s), `BREAKER_FALHAS_CONSECUTIVAS` (padrão 5), `BREAKER_INTERVALO_ABERTO` (padrão `30s`), `PAGINACAO_TAMANHO_PADRAO` (padrão 20), `PAGINACAO_TAMANHO_MAXIMO` (padrão 100), `HTTP_PORT` (padrão 8080), `OTEL_EXPORTER_OTLP_ENDPOINT`, `LOG_LEVEL` (padrão `info`).

**Alternatives considered**: Viper com arquivo de configuração (a ERS pede variáveis de ambiente; arquivo abriria caminho para segredo versionado); leitura preguiçosa por `os.Getenv` espalhada pelo código (impossível validar na largada e difícil de testar).
