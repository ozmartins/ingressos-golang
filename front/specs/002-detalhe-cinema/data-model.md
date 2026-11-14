# Data Model: Página de detalhes do cinema

Somente leitura; tudo vem do catálogo. Tipos novos em `src/api/tipos.ts` (já existem `Cinema`, `Filme`).

## Sala (API)
| Campo | Tipo | Observação |
|---|---|---|
| id | string (uuid) | |
| cinema_id | string (uuid) | vínculo com o cinema aberto |
| numero | number | único entre salas ativas do cinema |
| tipo_tela | `'2D' \| '3D' \| 'IMAX' \| 'VIP'` | |
| capacidade_total | number | exibida como "N lugares" |
| ativo | boolean | só ativas são listadas |

(`fileiras` é recebido mas ignorado pela página.)

## SessaoNaGrade (API)
`id`, `filme_id`, `filme_titulo`, `cinema_id`, `cinema_nome`, `sala_numero`, `tipo_tela`, `data_hora_inicio` (ISO, UTC), `idioma` (`DUBLADO|LEGENDADO`), `preco_base` (decimal em texto). A grade só devolve sessões agendadas/em andamento.

## Modelos de visão (calculados em `dominio/programacao.ts`)
- **SessaoExibida**: `id`, `inicio: Date`, `dia: string (AAAA-MM-DD local)`, `salaNumero`, `tipoTela`, `idioma`, `preco: number`.
- **ProgramacaoFilme**: `filme: Filme` (sempre em cartaz), `sessoes: SessaoExibida[]` ordenadas por `inicio`.
- **SalaComSessoes**: `sala: Sala`, `quantidadeSessoes: number` (para exibir "sem sessões programadas", FR-006/US2).
- **Programacao**: `{ filmes: ProgramacaoFilme[]; salas: SalaComSessoes[]; dias: string[] }`.

## Regras
- Só entram sessões cujo filme (por `filme_id`) existe na listagem com `status = EM_CARTAZ` (FR-013); as demais são descartadas antes de qualquer contagem, agrupamento ou lista de dias.
- Agrupamento por `filme_id`; filmes ordenados pela primeira sessão (depois por título); sessões por `inicio` (desempate por `salaNumero`).
- `dias` = dias locais distintos das sessões, ordenados; filtro `dia` (ou nenhum) restringe `filmes[].sessoes` e remove filmes que ficarem sem sessões.
- Contagem de sessões por sala usa todas as sessões (não o filtro de dia) para a seção de salas.
- Sessão cuja sala não está na lista de salas ativas continua exibida com o `sala_numero` da grade (se o filme estiver em cartaz).

## Estados da página
`carregando` → `pronto` | `nao-encontrado` | `erro`. Dentro de `pronto`: sem sessões / sem salas / filtro de dia sem resultado.

## Rota
`/cinemas/{id}` (id = uuid do cinema). Qualquer outro caminho = página inicial.
