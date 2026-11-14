import type { Poltrona } from '../api/tipos'

export type SituacaoAssento = 'livre' | 'selecionado' | 'ocupado' | 'indisponivel'

export interface FileiraMapa {
  letra: string
  poltronas: Poltrona[]
}

/** Fileiras em ordem alfabética (A primeiro) e assentos por número. */
export function agruparPorFileira(poltronas: Poltrona[]): FileiraMapa[] {
  const mapa = new Map<string, Poltrona[]>()
  for (const p of poltronas) {
    const lista = mapa.get(p.fileira)
    if (lista) lista.push(p)
    else mapa.set(p.fileira, [p])
  }
  return [...mapa.entries()]
    .sort(([a], [b]) => a.localeCompare(b, 'pt-BR'))
    .map(([letra, lista]) => ({ letra, poltronas: [...lista].sort((a, b) => a.numero - b.numero) }))
}

/** Rótulos das poltronas livres. */
export function rotulosLivres(poltronas: Poltrona[]): Set<string> {
  return new Set(poltronas.filter((p) => p.status === 'LIVRE').map((p) => p.rotulo))
}

/** RESERVADA e OCUPADA aparecem como ocupado; status desconhecido, como indisponível. */
export function situacaoDoAssento(poltrona: Poltrona, selecionados: readonly string[]): SituacaoAssento {
  if (selecionados.includes(poltrona.rotulo)) return 'selecionado'
  if (poltrona.status === 'LIVRE') return 'livre'
  if (poltrona.status === 'RESERVADA' || poltrona.status === 'OCUPADA') return 'ocupado'
  return 'indisponivel'
}

const ROTULO_SITUACAO: Record<SituacaoAssento, string> = {
  livre: 'livre',
  selecionado: 'selecionado',
  ocupado: 'ocupado',
  indisponivel: 'indisponível',
}

const ROTULO_TIPO = { NORMAL: '', PCD: ', acessível', NAMORADEIRA: ', namoradeira' } as const

/** "Fileira B, assento 4, livre, acessível". */
export function rotuloAcessivel(poltrona: Poltrona, situacao: SituacaoAssento): string {
  return `Fileira ${poltrona.fileira}, assento ${poltrona.numero}, ${ROTULO_SITUACAO[situacao]}${ROTULO_TIPO[poltrona.tipo] ?? ''}`
}
