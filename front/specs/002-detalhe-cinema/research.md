# Research: Página de detalhes do cinema

## 1. Roteamento
- **Decision**: rota `/cinemas/{id}` resolvida por `location.pathname` num hook `useRota`, com `pushState`/`popstate`; sem biblioteca de roteamento.
- **Rationale**: uma única rota nova; a feature 001 já sincroniza a aba via `pushState`. Evita dependência e mantém o bundle pequeno. URL própria atende FR-002 (compartilhar/recarregar).
- **Alternatives**: `react-router` (excessivo para 1 rota); query string `/?cinema=id` (URL menos canônica e mistura com `?aba=`).

## 2. Preservar filtros ao voltar (FR-009, SC-005)
- **Decision**: `PaginaInicial` fica montada e oculta (`hidden`) enquanto o detalhe aparece; só é montada na primeira vez que a rota da lista é visitada. A aba ativa continua na query string `?aba=`; o link "Voltar" usa `history.back()` quando há histórico interno, senão navega para `/?aba=cinemas`.
- **Rationale**: o estado dos filtros já vive em `useState` da página; manter montada é o mesmo padrão usado entre abas e dispensa persistência extra.
- **Alternatives**: `sessionStorage` (código extra, risco de dados obsoletos); filtros na URL (mudança maior na 001).

## 3. Fonte dos dados vinculados ao cinema
- **Decision**: 4 consultas em paralelo ao `catalogo`: `GET /cinemas/{id}`, `GET /salas?cinema_id={id}&ativo=true`, `GET /sessoes?cinema_id={id}` (paginado, reusa `buscarTodas`) e `GET /filmes` (já usado na 001; filtra os ids que aparecem nas sessões).
- **Rationale**: `SessaoNaGrade` já traz `filme_id`, `filme_titulo`, `sala_numero`, `tipo_tela`, `data_hora_inicio`, `idioma`, `preco_base` e o servidor já exclui canceladas/finalizadas (`statusVisiveis()`); falta só gênero/duração/classificação/imagem do filme, obtidos da listagem de filmes. Nenhuma mudança no backend.
- **Alternatives**: `GET /filmes/{id}` por filme (N+1 requisições); alterar a grade para incluir mais campos do filme (muda contrato do backend, fora de escopo).
- **Nota (FR-013)**: a página exibe apenas filmes `EM_CARTAZ`. Usa-se `buscarTodosFilmes` (sem o filtro da 001) e o domínio descarta sessões cujo filme não esteja em cartaz ou não conste da listagem. Por isso a listagem de filmes passa a ser obrigatória: se falhar, a programação mostra erro com "Tentar novamente" (não há degradação para só o título).

## 4. Cinema inexistente/inativo
- **Decision**: `GET /cinemas/{id}` 404 ou `ativo=false` → estado "cinema não encontrado" com link para a lista; demais falhas → erro com "Tentar novamente".
- **Rationale**: distingue ausência de falha transitória (FR-010). Id malformado (400) também cai em "não encontrado".

## 5. Filtro por dia e fuso horário
- **Decision**: filtrar no cliente pelo dia local do navegador, sobre todas as sessões já carregadas; opções de dia derivadas das sessões existentes.
- **Rationale**: o parâmetro `data` do servidor divide o dia em UTC e causaria divergência com o horário exibido (ex.: sessão 22h no Brasil cai no dia seguinte em UTC). Sem nova requisição ao trocar o dia.
- **Alternatives**: `?data=` do servidor (fuso incorreto, uma requisição por dia).

## 6. Formatação
- **Decision**: `Intl.DateTimeFormat('pt-BR')` para data/hora e `Intl.NumberFormat('pt-BR', {style:'currency', currency:'BRL'})` para preço (`preco_base` é texto decimal; converter com `Number`, aceitável para exibição).
- **Rationale**: FR-012; sem dependências.

## 7. Cartão clicável e acessível
- **Decision**: o `<li>` do cartão contém um `<a href="/cinemas/{id}">` cobrindo o card (título como âncora, `::after` expandindo a área clicável); clique simples (sem modificadores) é interceptado para navegação sem recarga.
- **Rationale**: teclado, abrir em nova aba e leitores de tela funcionam nativamente (FR-001).
