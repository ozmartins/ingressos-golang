import type { Filme, Idioma, Sala, SessaoNaGrade, TipoTela } from '../api/tipos'

export interface SessaoExibida {
  id: string
  filme: Filme
  inicio: Date
  /** Dia local no formato AAAA-MM-DD. */
  dia: string
  salaNumero: number
  tipoTela: TipoTela
  idioma: Idioma
  preco: number
}

export interface ProgramacaoFilme {
  filme: Filme
  sessoes: SessaoExibida[]
}

export interface SalaComSessoes {
  sala: Sala
  quantidadeSessoes: number
}

/** Dia no fuso local do navegador, como AAAA-MM-DD. */
export function diaLocal(data: Date): string {
  const mes = String(data.getMonth() + 1).padStart(2, '0')
  const dia = String(data.getDate()).padStart(2, '0')
  return `${data.getFullYear()}-${mes}-${dia}`
}

/**
 * Junta cada sessão ao seu filme e ordena cronologicamente. Só sobram sessões
 * de filmes em cartaz que constem da listagem (FR-013).
 */
export function montarSessoes(sessoes: SessaoNaGrade[], filmes: Filme[]): SessaoExibida[] {
  const emCartaz = new Map(filmes.filter((f) => f.status === 'EM_CARTAZ').map((f) => [f.id, f]))
  const exibidas: SessaoExibida[] = []
  for (const sessao of sessoes) {
    const filme = emCartaz.get(sessao.filme_id)
    if (!filme) continue
    const inicio = new Date(sessao.data_hora_inicio)
    exibidas.push({
      id: sessao.id,
      filme,
      inicio,
      dia: diaLocal(inicio),
      salaNumero: sessao.sala_numero,
      tipoTela: sessao.tipo_tela,
      idioma: sessao.idioma,
      preco: Number(sessao.preco_base),
    })
  }
  return exibidas.sort((a, b) => a.inicio.getTime() - b.inicio.getTime() || a.salaNumero - b.salaNumero)
}

/** Todas as salas ativas, por número, com a quantidade de sessões exibidas em cada uma. */
export function salasComSessoes(salas: Sala[], sessoes: SessaoExibida[]): SalaComSessoes[] {
  return [...salas]
    .sort((a, b) => a.numero - b.numero)
    .map((sala) => ({
      sala,
      quantidadeSessoes: sessoes.filter((s) => s.salaNumero === sala.numero).length,
    }))
}

/** Um bloco por filme (pela primeira sessão, depois pelo título), sessões cronológicas. */
export function agruparPorFilme(sessoes: SessaoExibida[]): ProgramacaoFilme[] {
  const grupos = new Map<string, ProgramacaoFilme>()
  for (const sessao of sessoes) {
    const grupo = grupos.get(sessao.filme.id)
    if (grupo) grupo.sessoes.push(sessao)
    else grupos.set(sessao.filme.id, { filme: sessao.filme, sessoes: [sessao] })
  }
  const ordenar = (a: SessaoExibida, b: SessaoExibida) =>
    a.inicio.getTime() - b.inicio.getTime() || a.salaNumero - b.salaNumero
  return [...grupos.values()]
    .map((g) => ({ ...g, sessoes: [...g.sessoes].sort(ordenar) }))
    .sort(
      (a, b) =>
        a.sessoes[0].inicio.getTime() - b.sessoes[0].inicio.getTime() ||
        a.filme.titulo.localeCompare(b.filme.titulo, 'pt-BR'),
    )
}

/** Dias locais distintos, em ordem. */
export function diasDisponiveis(sessoes: SessaoExibida[]): string[] {
  return [...new Set(sessoes.map((s) => s.dia))].sort()
}

/** `dia` vazio = sem restrição. */
export function filtrarPorDia(sessoes: SessaoExibida[], dia: string): SessaoExibida[] {
  return dia ? sessoes.filter((s) => s.dia === dia) : sessoes
}
