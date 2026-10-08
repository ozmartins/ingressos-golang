# Research: Roteamento do Estoque via chi

Cada decisão foi checada contra o código atual e contra um experimento executado
(chi v5.3.2 × `net/http.ServeMux` do Go 1.25, mesmo conjunto de rotas e requisições),
não contra a spec (princípio IX).

## D1 — Versão e superfície do chi

**Decisão**: `github.com/go-chi/chi/v5` **v5.3.2** (já no cache local de módulos; exige
Go 1.23, o `go.mod` está em 1.25). Usar apenas `chi.NewRouter`, `Get`/`Post`/`Handle`,
`MethodNotAllowed` e `middleware.GetHead`. Nada de `Route`/`Group`/`Mount`/`Use` extra.

**Rationale**: a v5.3.2 já chama `r.SetPathValue` (`mux.go:479`), então os handlers
continuam lendo `r.PathValue("sessao_id")` sem mudança. Princípio VII: sem
sub-roteadores nem middlewares do chi que ninguém pediu.

**Alternativas**: `chi.URLParam` nos handlers — rejeitada, muda código que já funciona.

## D2 — Diferenças medidas entre o ServeMux e o chi

Experimento com `POST /a/{id}/b`, `GET /a/{id}/c`, `GET /docs`, `GET /docs/`:

| Caso | ServeMux (hoje) | chi puro | Tratamento |
|------|-----------------|----------|------------|
| `GET` em rota só-`POST` | 405, `Allow: POST`, corpo `Method Not Allowed\n` (text/plain) | 405, `Allow: POST`, **corpo vazio**, sem Content-Type | D3 |
| `DELETE`/`OPTIONS` em rota `GET` | 405, `Allow: GET, HEAD` | 405, `Allow: GET` | D3 |
| `HEAD` em rota `GET` | 200 | **405** | D4 |
| `GET /nada` | 404 `404 page not found\n` text/plain | idêntico | — |
| `GET /docs/x` (padrão `GET /docs/` é subárvore) | 200 | `/docs/` casa só o exato → 404 | D5 |
| `GET /a/x%2Fy/c` | `PathValue` = `x/y` (decodificado) | `PathValue` = `x%2Fy` (cru) | D6 |
| `GET /a//x/c`, `/a/x/../x/c` | **307** redirecionando ao caminho limpo | 404 | D7 (levar ao mantenedor) |
| `GET /a/x/c/` | 404 | 404 | — |

## D3 — 405: corpo e `Allow`

**Decisão**: registrar `MethodNotAllowed` com handler próprio que escreve
`Method Not Allowed\n` (`http.Error`) e o cabeçalho `Allow` calculado. O chi **não expõe**
os métodos permitidos a um handler customizado (experimento: `Allow` vem vazio), então o
`Allow` é obtido sondando o próprio roteador com `Match` para os métodos HTTP comuns
(ordem alfabética, `HEAD` junto de `GET`, como o ServeMux).

**Rationale**: FR-005 manda preservar. A alternativa de deixar o 405 padrão do chi muda
corpo e `Allow` — rejeitada. A alternativa de embrulhar o `ResponseWriter` para consertar
o 405 padrão é mais frágil que sondar com `Match`.

**Risco**: a sondagem precisa ser coberta por teste de caracterização (ver D8).

## D4 — `HEAD`

**Decisão**: `middleware.GetHead` no roteador (encaminha `HEAD` à rota `GET` quando não há
rota `HEAD`). Verificado: `HEAD /b` → 200.

## D5 — `/docs/` como subárvore

**Decisão**: registrar `GET /docs` e `GET /docs/*`. Mantém `/docs/` e `/docs/qualquer`
servindo a UI, como hoje.

## D6 — Parâmetro de caminho codificado

**Decisão**: um helper privado no adaptador, `sessaoID(r)`, aplica `url.PathUnescape` ao
`r.PathValue("sessao_id")` (valor inválido → devolve o cru, que o caso de uso já rejeita
como identificador inválido). Os dois handlers passam a chamá-lo.

**Rationale**: preserva o valor que os casos de uso recebem hoje; mudança de 1 linha por
handler. `sessao_id` é UUID na prática, então o caso é teórico — mas FR-003 exige igualdade.

## D7 — Redirecionamento 307 de caminhos não canônicos

**Decisão**: **não replicar**; diferença conhecida e **aceita pelo mantenedor em 2026-10-07**
(FR-005, princípio X). Caminhos como `//` ou `..` hoje recebem 307 para o caminho limpo; com o chi
recebem 404. Nenhum cliente do repositório (catálogo, web, quickstart) gera esses caminhos.
Replicar exigiria `middleware.CleanPath` + redirecionamento próprio — complexidade sem
necessidade demonstrada (princípio VII).

## D8 — Estratégia de teste

**Decisão**: teste de caracterização escrito **antes** da troca e rodado contra o ServeMux
atual, depois contra o chi: uma tabela (método, caminho) → (status, `Allow`,
`Content-Type`, corpo) cobrindo as linhas da tabela D2 para a API REST e para a saúde.
Os testes existentes de `handlers_test.go` ficam inalterados (passam por `Rotas()`).
`health.go` hoje não tem teste: ganha o da tabela (princípio VIII: interface exposta).

## D9 — Onde mora a configuração comum

**Decisão**: a API REST e a saúde precisam do mesmo roteador configurado (GetHead + 405).
Um construtor `novoRoteador()` em `internal/adapter/http/roteador.go`, exportado como
`NovoRoteador()` para o `health` reutilizar. Sem pacote novo; `health` (plataforma) pode
importar o adaptador — a regra proíbe o contrário.

**Alternativa**: duplicar ~15 linhas no `health` — rejeitada: o `Allow` por sondagem é
sutil, duas cópias divergem.

## D10 — O que NÃO muda

`main.go` continua recebendo `http.Handler` (`apiREST.Rotas()`, `health.Handler()`), então
servidores, timeouts e encerramento ordenado não mudam (FR-009, FR-010). Contratos
(`openapi.yaml`, proto, AMQP) intocados — esta feature **não tem `contracts/`**.
