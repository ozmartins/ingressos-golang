# Quickstart: validar a página inicial

## Pré-requisitos
- Catálogo no ar na porta 8082 com dados: `cd ingressos-golang && docker compose up --build catalogo` (ou a plataforma inteira; ver `urls.txt`).
- Alguns filmes (incluindo um `BREVE` e um `FORA_DE_CARTAZ`) e cinemas (incluindo um inativo) cadastrados, via admin ou Swagger em `http://localhost:8082/docs`.

## Rodar
```
cd ingressos-golang/front
npm install
npm run dev        # abre http://localhost:5173
npm test           # lógica pura e componentes
npm run lint && npm run build
```

## Cenários de validação
1. Abrir `http://localhost:5173/` → aba **Filmes** ativa e lista carregada (US1-1). Filme fora de cartaz não aparece (FR-018).
2. Digitar parte de um título, com e sem acento/maiúsculas → lista reduz imediatamente; contagem atualiza (US1-2, FR-008/011).
3. Combinar gênero + classificação + situação → só itens que atendem a todos; "Limpar filtros" restaura (US1-4/5).
4. Ir à aba **Cinemas** → lista de cinemas ativos (US2-1). Filtrar por nome, estado e depois cidade; as cidades listadas pertencem ao estado (US2-2/3/4).
5. Filtrar sem resultado → mensagem "nenhum resultado" com atalho para limpar (edge case).
6. Aplicar filtros em Filmes, ir a Cinemas e voltar → filtros mantidos (US3-1). Recarregar em `/?aba=cinemas` → abre em Cinemas (US3-3).
7. Parar o catálogo e recarregar → mensagem de erro com "Tentar novamente"; subir o catálogo e tentar de novo → lista aparece (FR-012).
8. Teclado: Tab até as abas, setas/Home/End trocam, indicação visível da aba ativa (FR-015).
9. Janela de 360 px → sem rolagem horizontal (SC-007).

Contratos: [contracts/catalogo-consumido.md](contracts/catalogo-consumido.md). Entidades: [data-model.md](data-model.md).
