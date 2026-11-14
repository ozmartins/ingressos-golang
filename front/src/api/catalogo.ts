import type { Cinema, Filme, PaginaApi } from './tipos'

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
  const filmes = await buscarTodas<Filme>('/api/v1/filmes')
  return filmes.filter((filme) => filme.status !== 'FORA_DE_CARTAZ')
}

/** Somente cinemas ativos. */
export function buscarCinemas(): Promise<Cinema[]> {
  return buscarTodas<Cinema>('/api/v1/cinemas', { ativo: 'true' })
}
