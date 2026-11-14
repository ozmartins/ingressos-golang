# Research: Página de seleção de assentos

## R1 — Endereço da página e escolha da sessão
- **Decisão**: `/cinemas/{cinemaId}/filmes/{filmeId}/comprar?sessao={sessaoId}`. O botão (por filme) navega sem `sessao`; a página escolhe a próxima sessão futura do filme no cinema e passa a refletir a escolha em `?sessao=` (replaceState) para o link ser compartilhável.
- **Racional**: atende "endereço estável" (FR-001) e a premissa de sessão padrão; reusa `rotas.ts`.
- **Alternativas**: `/sessoes/{id}/assentos` (perde cinema/filme para o seletor e para o "voltar"); botão por sessão (muda a UI da página do cinema, fora do pedido).

## R2 — Guarda de login
- **Decisão**: `App` resolve a rota `compra`; se `useSessao().sessao === null` (inclui token expirado, pois `lerSessao` valida `expiraEm`), chama `urlAutenticacao('login', pathname+search)` e `location.assign(url)`; renderiza apenas "Redirecionando para o login…". O retorno já é tratado por `useSessao` (`navegar(r.retorno)`). Vale para clique e acesso direto.
- **Racional**: reuso total do fluxo PKCE de `BarraConta`; nenhum código de auth novo.
- **Alternativas**: modal de login (não existe); guardar o filme em estado global (não sobrevive ao redirect).
- **Cuidado**: disparar o redirect uma única vez (ref) para não repetir em re-render; `retorno` só aceita caminhos internos (começa com `/` e não `//`) — validar em `concluirAutenticacao`/`navegar`.

## R3 — Acesso ao estoque a partir do navegador
- **Decisão**: proxy Vite `/estoque` → `http://localhost:8085` (dev e preview), base configurável por `VITE_ESTOQUE_URL` (padrão `/estoque`), espelhando o que já existe para o catálogo.
- **Racional**: o estoque não envia CORS (nenhum código CORS no serviço); não mexer no backend.
- **Alternativas**: habilitar CORS no estoque (muda outro serviço; fora do escopo).

## R4 — Dados da sessão e do mapa
- **Decisão**: `catalogo`: `buscarCinema(id)`, `buscarTodosFilmes()` (acha o filme; imagem, classificação), `buscarSessoes(cinemaId)` filtrando `filme_id` e `data_hora_inicio` futura (já traz `sala_numero`, `tipo_tela`, `idioma`, `preco_base`). `estoque`: `GET /api/v1/sessoes/{id}/poltronas` (Bearer) → `{sessao_id, poltronas[{rotulo,fileira,numero,tipo,status}]}`.
- **Racional**: tudo já existe; sem endpoint novo. Trocar de sessão refaz só a consulta do mapa.
- **Alternativas**: `GET /sessoes/{id}` do catálogo (sem sala_numero/filme_titulo resolvidos → mais chamadas).

## R5 — Situações visuais do assento
- **Decisão**: mapeamento puro: `status` LIVRE→livre; RESERVADA e OCUPADA→"ocupado" (não selecionável; a API não distingue para o usuário); selecionado vem do estado local. `tipo` NORMAL/PCD/NAMORADEIRA→ícone/rótulo distinto. "Indisponível" do spec = sala/sessão que não aceita reserva (mapa inteiro desabilitado) — a API de poltronas não tem status "indisponível" por assento; assento sem status reconhecido é tratado como indisponível (defensivo).
- **Racional**: fiel ao contrato; acessível por texto (`aria-label` "Fileira B, assento 4, livre").

## R6 — Dinheiro e taxa
- **Decisão**: converter `preco_base` ("42.00") para centavos inteiros (parse de string, sem `parseFloat` em conta); total = centavos × qtd + taxa; taxa = 0 (sem fonte na plataforma), linha "Total taxa" mostra R$ 0,00. Exibir com `formatarPreco` existente (a estender para aceitar centavos).
- **Alternativas**: float (erro de arredondamento, viola SC-005).

## R7 — Reserva ("Continuar") e erros
- **Decisão**: `POST /api/v1/sessoes/{id}/bloqueios {poltronas_ids:[rotulos]}`. Tratamento: 201 → confirmação com `expira_em`; 409 → aviso, recarregar mapa, manter seleção ∩ livres; 400 com `metadata.limite` → aviso de limite; 401 → refazer login com retorno à mesma página, mantendo a seleção no `sessionStorage` por chave da sessão (não perde ao voltar do login); 404/422 → mensagem "sessão não aceita reserva"; 5xx/rede → erro com tentar novamente. Botão desabilitado durante a chamada (evita duplo envio).
- **Racional**: exclusividade e dupla venda são garantidas pelo PostgreSQL no estoque (CLAUDE.md); o front só reflete.

## R8 — Atualização e responsividade
- **Decisão**: sem polling (clarificação). Mapa em contêiner com rolagem horizontal própria (`overflow-x:auto`) e botões de assento ≥ 24 px com alvo de toque ampliado; resumo abaixo do mapa em telas estreitas, lateral (grid) ≥ 900 px. "Zoom" da referência fica fora (rolagem cobre o caso).

## R9 — Teste
- **Decisão**: Vitest com `fetch` injetado/mocado como nos testes existentes; lógica pura testada sem DOM; `PaginaCompra` com carregadores injetáveis (mesmo padrão `CarregadoresCinema`).
