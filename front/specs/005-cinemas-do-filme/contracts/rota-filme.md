# Contrato de UI: rota do filme

**Endereço**: `/filmes/{filmeId}` (`caminhoFilme(id)`; `lerRota` → `{ tipo: 'filme', id }`).

**Entrada**: card de filme na aba Filmes (link no título, card inteiro clicável, ativável por teclado) ou link direto.

**Consultas (públicas, sem alteração de contrato)**:
- `GET /api/v1/filmes/{id}` → 200 | 400/404 → "Filme não encontrado".
- `GET /api/v1/sessoes?filme_id={id}&page&page_size` (todas as páginas).
- `GET /api/v1/cinemas?ativo=true&page&page_size` (todas as páginas).

| Situação | Comportamento |
|----------|---------------|
| Carregando | `role="status"` "Carregando filme…" |
| Filme inexistente | "Filme não encontrado" + link para a aba Filmes |
| Falha de rede/HTTP | `role="alert"` + "Tentar novamente" |
| Sem cinemas com sessão futura | dados do filme + "Este filme não está em cartaz em nenhum cinema." |
| Filtros sem resultado | mensagem + "Limpar filtros" |
| Lista | contagem (`aria-live`), filtros UF/Cidade, itens com "Comprar Ingressos" |

**Item de cinema**: logo/marcador, nome, `cidade / UF`, endereço, botão "Comprar Ingressos" (`type="button"`) → `navegar(caminhoCompra(cinemaId, filmeId))`. Sem login a página de compra redireciona ao Keycloak e retorna a ela (comportamento da feature 003).

**Voltar**: "← Voltar aos filmes" → `history.back()` se veio do app, senão `/?aba=filmes`.

**Acessibilidade**: filtros em `<form role="search" aria-label="Filtrar cinemas">`; cada controle com `<label>`; botão com texto "Comprar Ingressos" e `aria-label` contendo o nome do cinema.
