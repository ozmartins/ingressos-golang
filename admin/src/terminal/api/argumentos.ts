import type { Definicao } from './tabela'

export interface ErroUso {
  erro: string
}

export interface Argumentos {
  id?: string
  query: Record<string, string>
}

export function ehErro(x: Argumentos | ErroUso): x is ErroUso {
  return 'erro' in x
}

export function destacar(texto: string): [string, string] {
  const t = texto.trim()
  const i = t.search(/\s/)
  if (i === -1) return [t, '']
  return [t.slice(0, i), t.slice(i).trim()]
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
    if (restante === '') return { id, query }
    return { erro: def.usaCorpo ? avisoSemJson(def) : `argumento inesperado: ${restante}` }
  }

  return resto.trim() === '' ? { query } : { erro: avisoSemJson(def) }
}

function avisoSemJson(def: Definicao): string {
  const comando = def.usaId ? `${def.controller} ${def.acao} <id>` : `${def.controller} ${def.acao}`
  return `este comando não recebe JSON nem campos na linha: digite apenas "${comando}" e responda aos campos pedidos`
}
