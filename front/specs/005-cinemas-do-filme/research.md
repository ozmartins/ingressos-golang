# Research: Cinemas com o filme em cartaz

## D1 — Fonte dos cinemas com o filme em cartaz
- **Decision**: `GET /api/v1/sessoes?filme_id={id}` (paginado, via `buscarTodas`) + `buscarCinemas()` (ativos) cruzados no cliente por `cinema_id`; só sessões com `data_hora_inicio` futura contam.
- **Rationale**: o filtro `filme_id` já existe no catálogo (openapi `/sessoes`) e `SessaoNaGrade` traz `cinema_id`; evita N consultas (uma por cinema) e não exige endpoint novo. Cinemas inativos ficam fora por `ativo=true`.
- **Alternatives**: chamar `buscarSessoes(cinemaId)` por cinema (N+1, lento); criar endpoint `/filmes/{id}/cinemas` no catálogo (fora de escopo, backend intocado).

## D2 — Dados do filme
- **Decision**: novo `buscarFilme(id)` com `GET /api/v1/filmes/{id}`; 404/400 → `FilmeNaoEncontrado` (mesmo padrão de `CinemaNaoEncontrado`).
- **Rationale**: a página precisa abrir por link direto/recarga (FR-002), sem depender da lista da aba.
- **Alternatives**: reaproveitar `buscarFilmes()` e filtrar (baixa a lista inteira e oculta fora de cartaz, que deve dar "sem cinemas" em vez de "não encontrado").

## D3 — Filme "em breve"/"fora de cartaz"
- **Decision**: a página abre normalmente e mostra "não está em cartaz em nenhum cinema". Não se filtra por `status` do filme na listagem; sem sessões futuras a lista fica vazia naturalmente, e o rótulo de situação aparece no cabeçalho.
- **Rationale**: cumpre o edge case da spec sem regra extra; a aba só mostra em cartaz e breve.

## D4 — Rota
- **Decision**: `/filmes/{id}` (`caminhoFilme`), interpretada por `lerRota` como `{ tipo: 'filme', id }`; id inválido na decodificação cai em `inicio` (como as demais).
- **Rationale**: simétrica a `/cinemas/{id}`, compartilhável e recarregável.

## D5 — Botão "Comprar Ingressos"
- **Decision**: `onClick → navegar(caminhoCompra(cinema.id, filme.id))`, igual a `PaginaCinema`. Login/retorno ficam em `PaginaCompra` (`useGuardaLogin` com retorno a pathname+search).
- **Rationale**: garante "mesmo comportamento" por reuso do mesmo caminho de código, sem duplicar a regra de login. A sessão padrão (próxima futura) é escolhida pela própria página de compra.

## D6 — Voltar
- **Decision**: link "← Voltar aos filmes" usa `history.back()` se `veioDoApp()`, senão `navegar('/?aba=filmes')`. Na compra, o "Voltar" existente já usa `history.back()` quando veio do app, retornando à página do filme; acesso direto e retorno do login (que recarrega a aplicação e perde o `history.state`) caem no cinema (comportamento atual preservado, fora do escopo desta feature).
- **Rationale**: FR-012 sem alterar `PaginaCompra`.

## D7 — Filtros
- **Decision**: reaproveitar `opcoesCinemas`/`ajustarCidade`/`filtrarCinemas` com `texto` sempre vazio; rótulo do campo "UF"; estado de filtros local à página (reinicia ao trocar de filme via `key`).
- **Rationale**: regra de dependência UF→cidade já testada.

## D8 — Card de filme clicável
- **Decision**: replicar o padrão de `CartaoCinema` (`cartao-clicavel`, link no título com `interceptarClique`).
- **Rationale**: consistência visual e de acessibilidade (teclado) com a feature 002.
