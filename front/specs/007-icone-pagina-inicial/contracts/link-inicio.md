# Contrato de UI: `LinkInicio` / `CabecalhoSite`

## CabecalhoSite
- Renderizado por `App` acima de qualquer rota; contém `LinkInicio` (esquerda) e `BarraConta` (direita).
- Presente em todas as rotas e estados (carregando, erro, não encontrado).

## LinkInicio
| Aspecto | Contrato |
|---------|----------|
| Elemento | `<a>` com `href="/"` |
| Papel/nome acessível | link "Página inicial" (`aria-label`), com `title="Página inicial"` |
| Conteúdo visual | SVG de casa, `aria-hidden="true"`, `currentColor`; sem texto visível |
| Clique simples | `preventDefault` + `navegar('/')` (sem recarregar); se `pathname === '/'`, apenas `preventDefault` |
| Ctrl/Cmd/Shift/Alt/botão do meio | comportamento padrão do navegador (nova aba/janela) |
| Teclado | alcançável por Tab; Enter aciona; `:focus-visible` com contorno `--foco` |
| Tamanho | área clicável ≥ 44×44 px |
| Cores | variáveis CSS existentes (`--texto`, `--destaque`, `--foco`); legível nos temas claro e escuro |
