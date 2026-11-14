export interface Historico {
  itens: string[]
  posicao: number
}

export interface Navegacao {
  historico: Historico
  valor: string
}

export function criar(): Historico {
  return { itens: [], posicao: 0 }
}

export function adicionar(h: Historico, comando: string): Historico {
  if (comando.trim() === '') return { ...h, posicao: h.itens.length }
  const itens = [...h.itens, comando]
  return { itens, posicao: itens.length }
}

export function anterior(h: Historico): Navegacao {
  if (h.itens.length === 0) return { historico: h, valor: '' }
  const posicao = Math.max(0, h.posicao - 1)
  return { historico: { ...h, posicao }, valor: h.itens[posicao] }
}

export function proximo(h: Historico): Navegacao {
  if (h.posicao >= h.itens.length - 1) {
    return { historico: { ...h, posicao: h.itens.length }, valor: '' }
  }
  const posicao = h.posicao + 1
  return { historico: { ...h, posicao }, valor: h.itens[posicao] }
}
