# Quickstart: validar a página de detalhes do cinema

## Pré-requisitos
- Plataforma no ar: `cd ingressos-golang && docker compose up --build` (catálogo em `localhost:8082`) com ao menos um cinema com salas e sessões (seeds do catálogo).
- `cd ingressos-golang/front && npm install`

## Automatizado
```
npm test        # filtros, programação, rotas e componentes
npm run lint
npm run build   # tsc -b + vite build
```

## Manual (`npm run dev`, abrir http://localhost:5173/?aba=cinemas)
1. Clicar num card de cinema → URL vira `/cinemas/{id}` e aparece nome, cidade/UF, endereço e logo (US1, FR-001/003).
2. Conferir que cada sessão mostra filme (título, gênero, duração, classificação, imagem), sala, horário, idioma e preço em R$ (FR-004/005/012) e que só aparecem sessões desse cinema (SC-002).
3. Conferir a lista de salas, incluindo uma sem sessões com a mensagem "sem sessões programadas" (US2).
4. Conferir agrupamento por filme, ordem cronológica e o filtro de dia (US3).
5. Aplicar um filtro na aba de cinemas, abrir um cinema e usar "Voltar" e o botão do navegador → filtros preservados (FR-009, SC-005).
6. Recarregar `/cinemas/{id}` e abrir em nova aba → página abre normalmente (FR-002).
7. Abrir `/cinemas/00000000-0000-0000-0000-000000000000` → "cinema não encontrado" com link para a lista.
8. Parar o catálogo e recarregar → mensagem de erro com "Tentar novamente".
9. Cinema sem sessões → dados do cinema + mensagem de ausência de sessões.
