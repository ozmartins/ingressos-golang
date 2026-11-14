# Research: Página inicial com abas de Filmes e Cinemas

Sem `NEEDS CLARIFICATION` pendentes. Decisões abaixo.

## 1. Origem dos dados e CORS
- **Decision**: Consumir `GET /api/v1/filmes` e `GET /api/v1/cinemas` do catálogo (públicos, `security: []`) por um prefixo `/catalogo` servido pelo proxy do Vite (dev e preview), com base configurável via `VITE_CATALOGO_URL` (padrão `/catalogo`).
- **Rationale**: O comentário em `admin/vite.config.ts` confirma que o catálogo não envia CORS; o `admin` já resolve assim. Reutiliza o padrão e mantém a mesma origem.
- **Alternatives**: Habilitar CORS no catálogo (altera outro serviço, fora do escopo); chamar `localhost:8082` direto (bloqueado pelo navegador).

## 2. Paginação do servidor
- **Decision**: Pedir `page_size=100` (máximo padrão) e seguir `pagina.tem_proxima` até esgotar, concatenando os itens.
- **Rationale**: Os filtros exigem o conjunto completo (opções derivadas de dados reais, FR-006); com ≤ 500 filmes são ≤ 5 requisições, dentro de SC-001. Se o servidor limitar abaixo de 100, usa-se o `tamanho` devolvido na resposta.
- **Alternatives**: Filtrar no servidor (só existem `status` e `ativo`; não há busca por texto/gênero/cidade); paginar na UI (conflita com contagem e opções de filtro globais).

## 3. Filtros no cliente
- **Decision**: Funções puras `filtrarFilmes(lista, filtros)` e `filtrarCinemas(lista, filtros)`; busca por texto com normalização NFD removendo diacríticos + `toLowerCase` + `trim`. Opções de gênero/classificação/estado/cidade extraídas do conjunto carregado, ordenadas com `localeCompare('pt-BR')`.
- **Rationale**: Testável sem DOM, resposta imediata (FR-008/009, SC-003).
- **Alternatives**: Debounce/consulta remota (desnecessário nesta escala).

## 4. Ocultar fora de cartaz e inativos
- **Decision**: Cinemas: pedir `ativo=true` ao servidor. Filmes: buscar tudo e descartar `FORA_DE_CARTAZ` no cliente (a API aceita um único `status`, e queremos EM_CARTAZ + BREVE).
- **Rationale**: Cumpre FR-018 com o menor número de requisições.
- **Alternatives**: Duas chamadas por status (mais requisições sem ganho).

## 5. Aba na URL
- **Decision**: Query string `?aba=filmes|cinemas` via `history.replaceState/pushState` e evento `popstate`; sem biblioteca de roteamento. Valor ausente/inválido cai em `filmes`.
- **Rationale**: Uma única página, dois valores; um router seria dependência sem necessidade (FR-014).
- **Alternatives**: React Router (excessivo); hash (menos limpo para compartilhar).

## 6. Preservar filtros ao trocar de aba
- **Decision**: Estado dos filtros no componente `PaginaInicial`, ambas as abas montadas (a inativa oculta com `hidden`) ou estado elevado; carregamento dos dados dispara uma vez por aba na primeira exibição.
- **Rationale**: FR-013 sem persistência em storage (spec: apenas visita corrente).
- **Alternatives**: localStorage (spec exclui persistência entre visitas).

## 7. Acessibilidade
- **Decision**: Padrão WAI-ARIA tabs: `role=tablist/tab/tabpanel`, `aria-selected`, `aria-controls`, setas esquerda/direita e Home/End movendo o foco; rótulos `<label>` nos filtros; contagem em região `aria-live="polite"`.
- **Rationale**: FR-015, SC-006.

## 8. Testes
- **Decision**: Vitest + jsdom + Testing Library; teste de lógica pura primeiro (TDD), depois componentes com `fetch` simulado.
- **Rationale**: Mesmo executor do `admin/`; o `front` hoje não tem runner de teste.
- **Alternatives**: Playwright E2E (útil, mas fica no quickstart manual para manter o escopo enxuto).

## 9. Estilo
- **Decision**: CSS simples em `index.css` com variáveis CSS, layout em grade responsiva (cartões), sem framework de UI.
- **Rationale**: Mantém dependências mínimas; atende SC-007 (≥ 360 px sem rolagem horizontal).
