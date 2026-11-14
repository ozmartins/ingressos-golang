# Quickstart: validar a página de seleção de assentos

## Pré-requisitos
- Plataforma de pé: `cd ingressos-golang && docker compose up --build` (catalogo :8082, estoque :8085, keycloak :8081). Dados de exemplo do catálogo carregados (seed) e sessões publicadas ao estoque (`make publicar-sessao` em `estoque/` se o mapa vier vazio/422).
- Front: `cd ingressos-golang/front && npm install && npm run dev`.

## Testes automatizados
```
npm test          # vitest run (domínio, componentes, rotas, guarda de login)
npm run lint
npm run build     # tsc -b && vite build
```

## Cenários manuais
1. **Deslogado**: abrir `/cinemas/{id}`, clicar "Comprar Ingressos" → Keycloak; entrar → cai em `/cinemas/{id}/filmes/{filme}/comprar` (não na home).
2. **Acesso direto deslogado**: colar a URL da compra em aba anônima → login → mesma página.
3. **Logado**: clique abre a página com cabeçalho (cinema, sala, dia, hora), mapa, legenda e resumo "0 itens / R$ 0,00", "Continuar" desabilitado.
4. Selecionar 2 assentos livres → resumo mostra 2 itens, assentos e total = 2 × preço; desmarcar um → 1.
5. Clicar em assento ocupado → nada acontece; selecionar o 11º → aviso de limite (10).
6. Alternar "Exibir números dos assentos".
7. Trocar de sessão → seleção zerada, mapa novo.
8. **Conflito**: em outra aba reservar o assento X (curl/POST bloqueios); na primeira, Continuar → aviso, mapa atualizado, demais assentos mantidos.
9. Continuar com sucesso → confirmação com assentos e prazo (~10 min).
10. Celular (360 px, devtools): sem rolagem horizontal da página; mapa rola dentro do contêiner; tema claro/escuro seguem os tokens.

Contratos: [rota-compra](contracts/rota-compra.md), [estoque-consumido](contracts/estoque-consumido.md). Modelo: [data-model](data-model.md).
