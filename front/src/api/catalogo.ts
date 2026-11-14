import type { Cinema, Filme, PaginaApi, Sala, SessaoNaGrade } from './tipos'

const TAMANHO_PAGINA = 100
const LIMITE_PAGINAS = 100

function base(): string {
  return (import.meta.env.VITE_CATALOGO_URL ?? '/catalogo').replace(/\/$/, '')
}

function ehPagina<T>(corpo: unknown): corpo is PaginaApi<T> {
  if (typeof corpo !== 'object' || corpo === null) return false
  const candidato = corpo as Partial<PaginaApi<T>>
  return Array.isArray(candidato.itens) && typeof candidato.pagina?.tem_proxima === 'boolean'
}

/** Lê todas as páginas de uma listagem pública do catálogo. */
async function buscarTodas<T>(caminho: string, extras: Record<string, string> = {}): Promise<T[]> {
  const itens: T[] = []
  for (let numero = 1; numero <= LIMITE_PAGINAS; numero++) {
    const parametros = new URLSearchParams({
      ...extras,
      page: String(numero),
      page_size: String(TAMANHO_PAGINA),
    })
    const resposta = await fetch(`${base()}${caminho}?${parametros}`)
    if (!resposta.ok) {
      throw new Error(`Falha ao consultar o catálogo (HTTP ${resposta.status}).`)
    }
    const corpo: unknown = await resposta.json()
    if (!ehPagina<T>(corpo)) {
      throw new Error('Resposta inesperada do catálogo.')
    }
    itens.push(...corpo.itens)
    if (!corpo.pagina.tem_proxima) return itens
  }
  return itens
}

/** Filmes em cartaz e em breve (fora de cartaz fica oculto). */
export async function buscarFilmes(): Promise<Filme[]> {
  const filmes = await buscarTodosFilmes()
  return filmes.filter((filme) => filme.status !== 'FORA_DE_CARTAZ')
}

/** Somente cinemas ativos. */
export function buscarCinemas(): Promise<Cinema[]> {
  return buscarTodas<Cinema>('/api/v1/cinemas', { ativo: 'true' })
}

/** Todos os filmes, inclusive fora de cartaz (usado para enriquecer sessões). */
export function buscarTodosFilmes(): Promise<Filme[]> {
  return buscarTodas<Filme>('/api/v1/filmes')
}

/** O cinema pedido não existe (ou o identificador é inválido). */
export class CinemaNaoEncontrado extends Error {
  constructor() {
    super('Cinema não encontrado.')
    this.name = 'CinemaNaoEncontrado'
  }
}

export async function buscarCinema(id: string): Promise<Cinema> {
  const resposta = await fetch(`${base()}/api/v1/cinemas/${encodeURIComponent(id)}`)
  if (resposta.status === 404 || resposta.status === 400) throw new CinemaNaoEncontrado()
  if (!resposta.ok) {
    throw new Error(`Falha ao consultar o catálogo (HTTP ${resposta.status}).`)
  }
  const corpo: unknown = await resposta.json()
  if (typeof corpo !== 'object' || corpo === null || typeof (corpo as Cinema).id !== 'string') {
    throw new Error('Resposta inesperada do catálogo.')
  }
  return corpo as Cinema
}

/** Salas ativas do cinema. */
export function buscarSalas(cinemaId: string): Promise<Sala[]> {
  return buscarTodas<Sala>('/api/v1/salas', { cinema_id: cinemaId, ativo: 'true' })
}

/** Sessões em exibição (agendadas ou em andamento) do cinema. */
export function buscarSessoes(cinemaId: string): Promise<SessaoNaGrade[]> {
  return buscarTodas<SessaoNaGrade>('/api/v1/sessoes', { cinema_id: cinemaId })
}
