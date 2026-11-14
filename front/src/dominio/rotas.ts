export type Rota =
  | { tipo: 'inicio' }
  | { tipo: 'cinema'; id: string }
  | { tipo: 'filme'; id: string }
  | { tipo: 'compra'; cinemaId: string; filmeId: string }
  | { tipo: 'pagamento'; cinemaId: string; filmeId: string }

export function caminhoCinema(id: string): string {
  return `/cinemas/${encodeURIComponent(id)}`
}

/** Página de um filme: cinemas que o têm em cartaz. */
export function caminhoFilme(id: string): string {
  return `/filmes/${encodeURIComponent(id)}`
}

/** Página de compra (seleção de assentos); `sessaoId` opcional vai na consulta. */
export function caminhoCompra(cinemaId: string, filmeId: string, sessaoId?: string): string {
  const base = `${caminhoCinema(cinemaId)}/filmes/${encodeURIComponent(filmeId)}/comprar`
  return sessaoId ? `${base}?sessao=${encodeURIComponent(sessaoId)}` : base
}

/** Página de pagamento com cartão (simulado), aberta após reservar os assentos. */
export function caminhoPagamento(cinemaId: string, filmeId: string): string {
  return `${caminhoCinema(cinemaId)}/filmes/${encodeURIComponent(filmeId)}/pagamento`
}

/**
 * Interpreta o `pathname`: `/cinemas/{id}`, `/filmes/{id}`, `/cinemas/{id}/filmes/{id}/comprar` e
 * `/cinemas/{id}/filmes/{id}/pagamento` são rotas próprias; o resto é a página inicial.
 */
export function lerRota(pathname: string): Rota {
  const partes = pathname.split('/').filter(Boolean)
  try {
    if (partes.length === 2 && partes[0] === 'cinemas') {
      return { tipo: 'cinema', id: decodeURIComponent(partes[1]) }
    }
    if (partes.length === 2 && partes[0] === 'filmes') {
      return { tipo: 'filme', id: decodeURIComponent(partes[1]) }
    }
    if (partes.length === 5 && partes[0] === 'cinemas' && partes[2] === 'filmes' && (partes[4] === 'comprar' || partes[4] === 'pagamento')) {
      const tipo = partes[4] === 'comprar' ? 'compra' : 'pagamento'
      return { tipo, cinemaId: decodeURIComponent(partes[1]), filmeId: decodeURIComponent(partes[3]) }
    }
  } catch {
    return { tipo: 'inicio' }
  }
  return { tipo: 'inicio' }
}
