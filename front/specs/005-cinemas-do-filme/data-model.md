# Data Model: Cinemas com o filme em cartaz

Sem persistência nova; apenas tipos/derivações no cliente. Tipos `Filme`, `Cinema`, `SessaoNaGrade` já existem em `src/api/tipos.ts`.

## Rota (`dominio/rotas.ts`)

| Variante | Campos | Endereço |
|----------|--------|----------|
| `{ tipo: 'filme' }` | `id: string` | `/filmes/{id}` |

## CinemaDoFilme (derivado)

| Campo | Origem |
|-------|--------|
| `cinema` | `Cinema` ativo |
| `proximaSessao` | menor `data_hora_inicio` futura do filme no cinema (informativo; ordenação interna) |
| `quantidadeSessoes` | número de sessões futuras do filme no cinema |

**Regra** `cinemasComOFilme(sessoes, cinemas, agora)`:
1. descarta sessões com início ≤ `agora`;
2. agrupa por `cinema_id`;
3. mantém só cinemas presentes em `cinemas` (ativos) com ≥ 1 sessão;
4. ordena por UF, cidade, nome (pt-BR).

## Filtros

`FiltrosCinemas` existente (`texto`, `estado`, `cidade`) com `texto = ''` fixo; `estado` exibido como "UF". Transição: trocar UF → `ajustarCidade` zera cidade incompatível; "Limpar filtros" → `filtrosCinemasVazios`.

## Estados da página

`carregando` → (`erro` | `filme não encontrado` | `pronto`); `pronto` → `sem cinemas` (lista vazia sem filtros) | `sem resultado` (lista vazia com filtros) | `lista`.
