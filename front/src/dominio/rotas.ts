export type Rota =
  | { tipo: 'inicio' }
  | { tipo: 'cinema'; id: string }
  | { tipo: 'compra'; cinemaId: string; filmeId: string }

export function caminhoCinema(id: string): string {
  return `/cinemas/${encodeURIComponent(id)}`
}

/** Página de compra (seleção de assentos); `sessaoId` opcional vai na consulta. */
export function caminhoCompra(cinemaId: string, filmeId: string, sessaoId?: string): string {
  const base = `${caminhoCinema(cinemaId)}/filmes/${encodeURIComponent(filmeId)}/comprar`
  return sessaoId ? `${base}?sessao=${encodeURIComponent(sessaoId)}` : base
}

/**
 * Interpreta o `pathname`: `/cinemas/{id}` e `/cinemas/{id}/filmes/{id}/comprar`
 * são rotas próprias; o resto é a página inicial.
 */
export function lerRota(pathname: string): Rota {
  const partes = pathname.split('/').filter(Boolean)
  try {
    if (partes.length === 2 && partes[0] === 'cinemas') {
      return { tipo: 'cinema', id: decodeURIComponent(partes[1]) }
    }
    if (partes.length === 5 && partes[0] === 'cinemas' && partes[2] === 'filmes' && partes[4] === 'comprar') {
      return { tipo: 'compra', cinemaId: decodeURIComponent(partes[1]), filmeId: decodeURIComponent(partes[3]) }
    }
  } catch {
    return { tipo: 'inicio' }
  }
  return { tipo: 'inicio' }
}
