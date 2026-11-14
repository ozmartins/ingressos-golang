# Tasks: URL do logo do cinema

**Input**: `specs/001-url-logo-cinema/` (plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md)

**Tests**: incluídos, pois o serviço já tem suítes de domínio, contrato e integração que cobrem o cinema e os requisitos FR-003/FR-004 exigem verificação.

Caminhos relativos a `ingressos-golang/catalogo/`. Formato: `- [ ] ID [P] [Story] Descrição com caminho`.

## Phase 1: Setup

- [x] T001 Confirmar a base verde antes de mexer: rodar `make test` e `make lint` em `ingressos-golang/catalogo/` e registrar o estado inicial

## Phase 2: Foundational (bloqueia todas as histórias)

- [x] T002 Criar `migrations/000007_logo_do_cinema.up.sql` com `ALTER TABLE catalogo.cinemas ADD COLUMN logo_url VARCHAR(2048)` (nullable, sem default), com comentário em português no estilo de `000003_ativar_cinemas.up.sql`
- [x] T003 [P] Criar `migrations/000007_logo_do_cinema.down.sql` com `ALTER TABLE catalogo.cinemas DROP COLUMN IF EXISTS logo_url`
- [x] T004 [P] Adicionar `LogoURL string` em `Cinema` e `DadosCinema` e a validação em `NovoCinema` (aparar; vazio = sem logo; `net/url`: esquema http/https, host presente, sem espaços, ≤ 2048; erros `shared.ErrValidacao` citando `logo_url`) em `internal/domain/catalogo/cinema.go`
- [x] T005 [P] Adicionar `LogoURL *string \`gorm:"column:logo_url"\`` em `cinemaRow` em `internal/adapter/postgres/modelos.go`
- [x] T006 [P] Adicionar `logo_url` aos schemas `Cinema` (string, `nullable: true`, `format: uri`, `maxLength: 2048`, **fora de `required`**) e `CinemaEntrada` (opcional, `nullable: true`, `maxLength: 2048`, mantendo `additionalProperties: false`) em `internal/adapter/http/openapi/openapi.yaml`, conforme `contracts/cinema-logo-url.md`

**Checkpoint**: migração, domínio, modelo de linha e contrato prontos.

## Phase 3: User Story 1 - Informar o logo ao cadastrar ou editar (P1) 🎯 MVP

**Objetivo**: gravar, alterar e remover a URL do logo; rejeitar URLs inválidas.

**Teste independente**: POST com URL válida → GET devolve a mesma; PUT altera; PUT sem o campo remove; URL inválida → 400.

- [x] T007 [P] [US1] Criar `internal/domain/catalogo/cinema_test.go` (arquivo novo) com testes de validação cobrindo: http válida, https válida, espaços nas pontas (aparada), vazia/só espaços (sem logo), `javascript:alert(1)`, `ftp://x`, `https://` (sem host), `https://a b.com` (espaço interno), texto livre, 2048 caracteres (aceita) e 2049 (rejeita); em cada caso inválido, afirmar `errors.Is(err, shared.ErrValidacao)` e que a mensagem contém `logo_url` e o motivo (esquema, formato ou tamanho)
- [x] T008 [P] [US1] Adicionar casos de contrato: POST/PUT com `logo_url` válida, PUT sem o campo remove o logo, URLs inválidas → 400 em `test/contract/cinemas_test.go`, verificando que o Problem Details cita o campo `logo_url`
- [x] T009 [US1] Ler `logo_url` em `cinemaEntradaDTO` (`*string`) e repassá-lo em `paraDadosCinema()` em `internal/adapter/http/dto.go`
- [x] T010 [US1] Gravar `logo_url` em `Criar` e `Atualizar` (vazio → `NULL`) em `internal/adapter/postgres/cinema_repository.go`
- [x] T011 [US1] Teste de integração de ida e volta no banco (criar com logo, atualizar, remover) em `test/integration/catalogo_test.go`

## Phase 4: User Story 2 - Consultar o logo nas respostas (P2)

**Objetivo**: devolver `logo_url` (ou `null`) em GET /cinemas, GET /cinemas/{id}, POST e PUT.

**Teste independente**: listar e detalhar cinemas com e sem logo e conferir o campo.

- [x] T012 [US2] Incluir `LogoURL *string \`json:"logo_url"\`` em `cinemaDTO` e preenchê-lo em `paraCinemaDTO` (vazio → `null`) em `internal/adapter/http/dto.go`
- [x] T013 [US2] Mapear `LogoURL` (NULL → vazio) em `cinemaRow.paraDominio` em `internal/adapter/postgres/cinema_repository.go`
- [x] T014 [P] [US2] Estender `test/contract/cinemas_test.go`: resposta sempre contém a chave `logo_url`, `null` sem logo e o valor com logo, em listagem e detalhe

## Phase 5: User Story 3 - Cinemas existentes continuam funcionando (P1)

**Objetivo**: nenhuma regressão em dados e clientes antigos.

**Teste independente**: com cinemas anteriores à migração, tudo segue acessível e editável, com `logo_url: null`.

- [x] T015 [US3] Garantir que fixtures e inserts existentes (`test/fixtures/catalogo_exemplo.sql`, `test/integration/performance_test.go`) continuam válidos sem a coluna nova e adicionar asserção de `logo_url` nulo para cinema legado em `test/integration/catalogo_test.go`
- [x] T016 [US3] Validar reversibilidade: `make migrate-up` seguido de `make migrate-down` e `make migrate-up` novamente, sem erro e sem perder os demais dados

## Phase 6: Polish

- [ ] T017 [P] Rodar `make test`, `make test-integration` e `make lint` e corrigir pendências (incluindo `depguard` do núcleo)
- [ ] T018 [P] Percorrer `quickstart.md` ponta a ponta contra o stack do `docker-compose.yml`
- [x] T019 [P] Mencionar o campo `logo_url` no `README.md` do catálogo, se ele descrever os campos do cinema

## Dependências e ordem

- Phase 2 bloqueia tudo. T002→T003 independentes; T004, T005, T006 paralelos entre si.
- US1 (T007–T011) e US2 (T012–T014) tocam os mesmos arquivos `dto.go` e `cinema_repository.go`: executar US1 antes de US2 (T009 antes de T012; T010 antes de T013).
- US3 depende de Phase 2 e de T010/T013. Polish por último.

## Paralelismo

- Phase 2: T003, T004, T005, T006 juntos.
- US1: T007 e T008 juntos (arquivos distintos), antes das implementações.
- Polish: T017, T018, T019.

## Estratégia

MVP = Phase 2 + US1 (gravar e validar o logo). Depois US2 (expor nas respostas) e US3 (verificação de legado/reversão). Observação: sem US2 o dado é gravado mas não retorna na API, então entregar US1+US2 juntos é o incremento realmente útil.

## Notas de execução

- T016 verificado aplicando up → down → up da 000007 em um Postgres 16 descartável via `psql` (o CLI `migrate` não está instalado nesta máquina).
- T017 parcial: `go vet`, `make test` e os testes de integração passam; `golangci-lint` não está instalado, então o `make lint` (inclui o `depguard`) **não foi executado**.
- T018 não executado: o stack do `docker-compose.yml` não foi subido; o quickstart foi coberto pelos testes de contrato e integração.
- Registrada a migração 000007 no harness `test/integration/main_test.go` (lista fixa de arquivos).
- Extra, fora do escopo original (pedido do usuário): imagens em `imagens-cinemas/`, rota estática `GET /imagens/*`, `IMAGENS_DIR`, `COPY` no Dockerfile e seeds regenerados com `logo_url`/`imagem_url` apontando para a nova pasta.
