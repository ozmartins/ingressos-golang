# Front: página inicial de Ingressos

Aplicação React + TypeScript + Vite voltada ao público. A página inicial tem duas abas, **Filmes** e **Cinemas**, cada uma com filtros para localizar rapidamente o item procurado.

- **Filmes:** busca por título e filtros de gênero, classificação etária e situação (em cartaz, em breve).
- **Cinemas:** busca por nome e filtros de estado e cidade (as cidades dependem do estado escolhido).
- A aba ativa fica na URL (`/?aba=filmes` ou `/?aba=cinemas`) e os filtros de cada aba são mantidos ao alternar entre elas.

- **Cinemas do filme (`/filmes/{id}`):** clicar no card de um filme abre a página com os dados do filme e os cinemas ativos que o têm em cartaz (ao menos uma sessão futura), com filtro por UF e cidade. Cada cinema tem o botão "Comprar Ingressos", com o mesmo comportamento do botão da página do cinema (login se necessário e abertura da compra do filme naquele cinema). Usa `GET /api/v1/filmes/{id}` e `/sessoes?filme_id=`. "Voltar" retorna à aba de filmes com os filtros preservados. Detalhes em `specs/005-cinemas-do-filme/`.
- **Detalhe do cinema (`/cinemas/{id}`):** clicar no card de um cinema abre a página com os dados do cinema, suas salas ativas e a programação (sessões agrupadas por filme, com filtro por dia). Só aparecem filmes **em cartaz**. "Voltar" retorna à aba de cinemas com os filtros preservados. Em hospedagem estática, `/cinemas/*` deve cair no `index.html` (o Vite dev/preview já faz isso).

Especificação, plano e tarefas: [`specs/001-home-filmes-cinemas/`](specs/001-home-filmes-cinemas/) e [`specs/002-detalhe-cinema/`](specs/002-detalhe-cinema/).

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

O `estoque` (mapa de assentos e reservas) também não envia CORS: `/estoque/*` é encaminhado para `http://localhost:8085/*`; base configurável por `VITE_ESTOQUE_URL` (padrão: `/estoque`).

### Página de compra

O botão "Comprar Ingressos" abre `/cinemas/{cinemaId}/filmes/{filmeId}/comprar?sessao={id}`. A rota exige login: sem sessão, o usuário vai ao Keycloak e volta à mesma página. Selecionar assentos e "Continuar" cria a reserva temporária no `estoque` (requer o serviço no ar na porta 8085). Detalhes em `specs/003-selecao-assentos/`.

Os dados são lidos das listagens públicas `GET /api/v1/filmes` e `GET /api/v1/cinemas`, percorrendo todas as páginas (100 itens por página). A página do cinema também usa `GET /api/v1/cinemas/{id}`, `/salas?cinema_id=`, `/sessoes?cinema_id=`. Filmes fora de cartaz e cinemas inativos não são exibidos. As imagens vêm de URLs remotas devolvidas pela API: `imagem_url` nos filmes e `logo_url` nos cinemas (sem URL, um substituto neutro é exibido).

## Estrutura

```
src/
├── api/          cliente do catálogo e tipos
├── dominio/      funções puras: normalização de texto, filtros, formatação
├── hooks/        useLista (carga/erro) e useAbaNaUrl
└── componentes/  PaginaInicial, Abas, AbaFilmes, AbaCinemas, cartões e estados
```
