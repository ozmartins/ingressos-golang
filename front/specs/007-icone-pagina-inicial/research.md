# Research: Ícone para retornar à página inicial

## D1 — Onde colocar o ícone
- **Decisão**: no cabeçalho renderizado por `App` (acima das rotas), à esquerda; `BarraConta` permanece à direita.
- **Racional**: `BarraConta` já aparece em todas as rotas e estados (carregando, erro, não encontrado), então um cabeçalho ali cumpre FR-001/002/009 sem editar cada página.
- **Alternativas**: ícone dentro de cada `Pagina*` (repetição, esquece páginas novas); menu/layout por rota (complexidade sem ganho).

## D2 — Ícone
- **Decisão**: SVG inline de casa (`aria-hidden`), com `aria-label="Página inicial"` e `title` no link.
- **Racional**: sem dependência nova, herda `currentColor` (contraste nos dois temas), dica ao passar o mouse e nome acessível.
- **Alternativas**: biblioteca de ícones (dependência por 1 ícone); emoji 🏠 (aparência varia por sistema, contraste não controlável).

## D3 — Navegação
- **Decisão**: `<a href="/">` com `interceptarClique` + `navegar('/')`; se já em `/`, só impede o padrão.
- **Racional**: reaproveita o padrão dos demais links (sem recarregar, Ctrl/Cmd/meio abrem nova aba, destino copiável, botão "voltar" do navegador funciona). A página inicial fica montada/oculta em `App`, então o estado (aba, filtros) é preservado.
- **Alternativas**: `history.back()` (não leva ao início); recarga completa `location.assign('/')` (perde estado e é mais lenta).

## D4 — Leiaute e acessibilidade
- **Decisão**: cabeçalho flex com `justify-content: space-between`; link com `min-width/min-height: 44px`, ícone centralizado, `:focus-visible` com o `outline` de `--foco` já usado no site; cores por variáveis existentes.
- **Racional**: atende FR-005/006/007 sem novos tokens; em telas estreitas os botões de conta já são compactos e o cabeçalho quebra linha com `flex-wrap` se faltar espaço.

## Itens NEEDS CLARIFICATION
Nenhum.
