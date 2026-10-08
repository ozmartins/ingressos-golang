# Research: Roteamento do Catálogo via chi

Cada decisão foi checada contra o código atual (`internal/adapter/http/`,
`test/contract/`) e contra um programa de comparação executado com o `chi` v5.3.2 já
presente no cache de módulos local — não contra a spec 001 (princípio IX).

## D1 — Qual chi e como entra

**Decisão**: `github.com/go-chi/chi/v5` v5.3.2 (`go 1.23` no módulo dele; o catálogo é
Go 1.25). Só o roteador: não entra `chi/middleware` além de `GetHead` (D4).

**Rationale**: v5.3.2 chama `r.SetPathValue` para cada parâmetro de rota
(`mux.go:479`), então os 11 pontos de `r.PathValue("id")` em `handlers.go` **continuam
valendo sem edição**. Confirmado executando: `/a/%C3%A7` entrega `"ç"` nos dois roteadores.

**Alternativas**: `chi.URLParam(r, "id")` nos handlers — rejeitada: edita 11 linhas e acopla
os handlers ao chi sem ganho (princípio VII).

## D2 — Tabela declarativa de rotas fica

**Decisão**: `rotas []Rota`, `Rota` e `Rotas()` não mudam. `NovoRouter` troca só o laço de
registro: `mux.Handle(m+" "+c, h)` → `r.Method(m, c, h)`. A sintaxe `{id}` é a mesma, então
nenhum `Caminho` muda, e `docs_test.go` (paridade com o OpenAPI) segue valendo sem edição.

**Rationale**: a spec (FR-008) exige uma única fonte das rotas; a tabela já cumpre isso.

## D3 — Middlewares globais: de `Encadear` para `r.Use`, e por quê (achado)

**Decisão**: `Telemetria`, `Recuperacao` e `Log` passam a ser registrados com
`r.Use(...)`, na mesma ordem (Telemetria → Recuperacao → Log); `Encadear` deixa de ser usada
em `NovoRouter`. `r.Use` do chi envolve também 404 e 405, então FR-005 continua valendo.

**Por que não manter `Encadear(mux, ...)` por fora**: o chi, ao receber a requisição, cria
uma cópia dela (`r.WithContext`) para guardar o contexto de rota, e é nessa cópia que grava
`r.Pattern`. Um middleware **fora** do chi lê `r.Pattern` vazio. Verificado:

```
chi: Pattern visto por fora = ""
std: Pattern visto por fora = "GET /a/{id}"
```

`middleware.Log` usa `r.Pattern` como rótulo `rota` de log e de métrica
(`middleware.go:52`) e cai para `r.URL.Path` quando vazio — com o chi por fora, **todo
identificador viraria um rótulo distinto** (cardinalidade explosiva na métrica
`HTTPDuracao`). Dentro do `r.Use`, os middlewares compartilham a mesma cópia em que o chi
grava o padrão, e o rótulo sai preenchido.

## D4 — Preservar `r.Pattern` com método e o HEAD (decisão do mantenedor, 2026-10-07)

Diferenças medidas entre `http.ServeMux` e chi v5.3.2:

| Caso | ServeMux (hoje) | chi (padrão) |
|---|---|---|
| `r.Pattern` | `GET /a/{id}` | `/a/{id}` (sem método) |
| `HEAD` em rota só-GET | 200 (roda o handler) | 405 |
| `POST` em rota só-GET/PUT | 405, `Allow: GET, HEAD, PUT`, corpo `Method Not Allowed` | 405, `Allow: GET` (incompleto), corpo vazio |
| 404 | `404 page not found` | igual |

**Decisão (mantenedor: "Preservar")**:

1. **Pattern**: cada handler registrado é embrulhado por uma função que faz
   `r.Pattern = rota.Metodo + " " + rota.Caminho` antes de chamar o handler. Mantém os rótulos
   de log, métrica e (indiretamente) o contrato operacional como estão.
2. **HEAD**: `r.Use(chimw.GetHead)` (último da cadeia) — HEAD cai na rota GET, como no ServeMux.
3. **405**: `r.MethodNotAllowed(h)` com `h` próprio: monta `Allow` perguntando ao próprio
   roteador (`rctx.Routes.Match(chi.NewRouteContext(), m, path)` para GET, POST, PUT, DELETE),
   acrescenta `HEAD` quando GET casa, ordena, e responde `http.Error(w, "Method Not Allowed", 405)`.
   O `Match` evita reimplementar casamento de padrão e evita uma segunda fonte de verdade.
4. **404**: o padrão do chi (`http.NotFound`) já é idêntico; nada a fazer.

**Alternativas**: aceitar o padrão do chi (rejeitada pelo mantenedor: mudaria rótulos de
métrica e o HEAD); calcular `Allow` a partir da tabela `rotas` (rejeitada: reimplementaria o
casamento de `{id}`).

## D5 — Divergências residuais que o chi não permite preservar a custo razoável

Levadas ao mantenedor (FR-006). **Decisão (2026-10-08): aceitar o comportamento do chi**
nos dois casos, fixado por `TestCaminhoNaoCanonicoResponde404` e
`TestIdComBarraCodificadaDaAMesmaRecusaQueIdInvalido` em `router_test.go`:

- **Limpeza de caminho**: `//` e `/../` — o ServeMux responde `307` para o caminho limpo; o
  chi responde `404`. Nenhum cliente do catálogo (frontend, estoque) gera esses caminhos.
- **`%2F` em parâmetro**: o ServeMux entrega o valor decodificado (`x/y`); o chi entrega o
  bruto (`x%2Fy`). Para um `{id}` inválido, o resultado esperado é o mesmo erro de validação
  nos dois casos; a tarefa de teste confirma isso em vez de supor.

## D6 — Fronteira do núcleo

**Decisão**: adicionar `github.com/go-chi` à regra `nucleo-sem-adaptadores` do depguard em
`.golangci.yml`, ao lado de `net/http`, `gorm.io` etc. (FR-009, SC-004).

## D7 — Como provar equivalência (princípio VIII e SC-002)

**Decisão**: um teste novo em `internal/adapter/http` percorre `Rotas()` e, para cada rota
documentada, verifica (a) o handler é atingido, (b) `r.PathValue("id")` chega, (c) rota
protegida sem credencial devolve a recusa padrão. Mais testes de borda para D4/D5: HEAD,
405 com `Allow` completo e corpo, 404, `/docs/`, e `r.Pattern` com método **visto pelo
`Log`** (o caso do achado D3). Os testes de contrato e de integração existentes passam sem
alterar asserções.
