# Contrato de UI: rota de compra

| Item | Valor |
|------|-------|
| Caminho | `/cinemas/{cinemaId}/filmes/{filmeId}/comprar` |
| Consulta | `?sessao={sessaoId}` (opcional; ausente/inválida → próxima sessão futura) |
| Acesso | Exige login. Sem sessão válida → Keycloak (`urlAutenticacao('login', caminho+consulta)`), depois volta à mesma URL |
| Origem | Botão "Comprar Ingressos" do card do filme em `/cinemas/{id}` → `navegar(caminhoCompra(cinemaId, filmeId))` (empilha histórico; "Voltar" retorna ao cinema) |
| Rota desconhecida/IDs malformados | `lerRota` devolve `inicio` (como hoje) |

## Funções (`dominio/rotas.ts`)
- `caminhoCompra(cinemaId, filmeId, sessaoId?)` → string com `encodeURIComponent`.
- `lerRota(pathname)` → `{ tipo: 'compra', cinemaId, filmeId }` para 4 segmentos `cinemas/*/filmes/*` + `comprar` (5 segmentos); `sessao` lido de `location.search` na página.

## Contrato de acessibilidade
- Assento: `button` com `aria-pressed` (selecionado) e `aria-label` "Fileira B, assento 4, livre|ocupado|acessível…"; ocupados `disabled`.
- Estados de carga/erro em `role="status"`/`role="alert"`.
