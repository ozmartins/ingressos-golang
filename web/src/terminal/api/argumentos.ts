import type { Definicao } from './tabela'

export interface ErroUso {
  erro: string
}

export interface Argumentos {
  id?: string
  query: Record<string, string>
  corpo?: unknown
}

export function ehErro(x: Argumentos | ErroUso | { corpo: unknown }): x is ErroUso {
  return 'erro' in x
}

export function destacar(texto: string): [string, string] {
  const t = texto.trim()
  const i = t.search(/\s/)
  if (i === -1) return [t, '']
  return [t.slice(0, i), t.slice(i).trim()]
}

export function lerCorpo(texto: string): { corpo: unknown } | ErroUso {
  if (texto.trim() === '') return { erro: 'corpo JSON ausente' }
  try {
    return { corpo: JSON.parse(texto) as unknown }
  } catch (e) {
    return { erro: `JSON inválido: ${e instanceof Error ? e.message : String(e)}` }
  }
}

export function lerArgumentos(def: Definicao, resto: string): Argumentos | ErroUso {
  const query: Record<string, string> = {}

  if (def.acao === 'listar') {
    for (const token of resto.split(/\s+/).filter(Boolean)) {
      const i = token.indexOf('=')
      if (i <= 0 || i === token.length - 1) {
        return { erro: `argumento inválido: ${token} (use chave=valor)` }
      }
      const chave = token.slice(0, i)
      if (!def.filtros.includes(chave)) {
        return { erro: `filtro desconhecido: ${chave} (filtros: ${def.filtros.join(', ')})` }
      }
      query[chave] = token.slice(i + 1)
    }
    return { query }
  }

  if (def.acao === 'health') {
    return resto.trim() === '' ? { query } : { erro: 'health não recebe argumentos' }
  }

  if (def.usaId) {
    const [id, restante] = destacar(resto)
    if (id === '') return { erro: 'identificador ausente' }
    if (def.usaCorpo) {
      const c = lerCorpo(restante)
      return ehErro(c) ? c : { id, query, corpo: c.corpo }
    }
    return restante === '' ? { id, query } : { erro: `argumento inesperado: ${restante}` }
  }

  const c = lerCorpo(resto)
  return ehErro(c) ? c : { query, corpo: c.corpo }
}
