export type Rota = { tipo: 'inicio' } | { tipo: 'cinema'; id: string }

export function caminhoCinema(id: string): string {
  return `/cinemas/${encodeURIComponent(id)}`
}

/** Interpreta o `pathname`: só `/cinemas/{id}` é rota própria; o resto é a página inicial. */
export function lerRota(pathname: string): Rota {
  const partes = pathname.split('/').filter(Boolean)
  if (partes.length === 2 && partes[0] === 'cinemas') {
    try {
      return { tipo: 'cinema', id: decodeURIComponent(partes[1]) }
    } catch {
      return { tipo: 'inicio' }
    }
  }
  return { tipo: 'inicio' }
}
