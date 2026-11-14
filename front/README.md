# Front: página inicial de Ingressos

Aplicação React + TypeScript + Vite voltada ao público. A página inicial tem duas abas, **Filmes** e **Cinemas**, cada uma com filtros para localizar rapidamente o item procurado.

- **Filmes:** busca por título e filtros de gênero, classificação etária e situação (em cartaz, em breve).
- **Cinemas:** busca por nome e filtros de estado e cidade (as cidades dependem do estado escolhido).
- A aba ativa fica na URL (`/?aba=filmes` ou `/?aba=cinemas`) e os filtros de cada aba são mantidos ao alternar entre elas.

Especificação, plano e tarefas: [`specs/001-home-filmes-cinemas/`](specs/001-home-filmes-cinemas/).

## Como rodar

Pré-requisito: o serviço `catalogo` no ar na porta 8082 (`cd .. && docker compose up --build catalogo`).

```
npm install
npm run dev       # http://localhost:5173
npm test          # testes (Vitest)
npm run lint
npm run build
```

## Acesso ao catálogo

O catálogo não envia cabeçalhos CORS. Em desenvolvimento (`npm run dev`) e no `npm run preview`, o Vite encaminha `/catalogo/*` para `http://localhost:8082/*` (ver `vite.config.ts`). Para apontar para outra base, defina `VITE_CATALOGO_URL` (padrão: `/catalogo`).

Os dados são lidos das listagens públicas `GET /api/v1/filmes` e `GET /api/v1/cinemas`, percorrendo todas as páginas (100 itens por página). Filmes fora de cartaz e cinemas inativos não são exibidos.

## Estrutura

```
src/
├── api/          cliente do catálogo e tipos
├── dominio/      funções puras: normalização de texto, filtros, formatação
├── hooks/        useLista (carga/erro) e useAbaNaUrl
└── componentes/  PaginaInicial, Abas, AbaFilmes, AbaCinemas, cartões e estados
```
