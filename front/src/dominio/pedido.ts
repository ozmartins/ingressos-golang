/** Máximo de assentos por pedido (`POLTRONAS_MAX_POR_BLOQUEIO` do estoque). */
export const LIMITE_ASSENTOS = 10

/** "42.00" → 4200; "42" → 4200; "42.5" → 4250. Sem ponto flutuante. */
export function precoEmCentavos(preco: string): number {
  const [inteiro, fracao = ''] = preco.trim().split('.')
  const centavos = (fracao + '00').slice(0, 2)
  return Number(inteiro) * 100 + Number(centavos)
}

export type ResultadoAlternar =
  | { rotulos: string[]; aviso?: undefined }
  | { rotulos: string[]; aviso: 'limite' }

/**
 * Marca/desmarca um assento. Só assentos livres podem ser marcados e o total
 * não passa de `limite`; ao recusar, devolve a seleção intacta (com `aviso` se foi o limite).
 */
export function alternarAssento(
  rotulos: string[],
  rotulo: string,
  livres: ReadonlySet<string>,
  limite: number = LIMITE_ASSENTOS,
): ResultadoAlternar {
  if (rotulos.includes(rotulo)) return { rotulos: rotulos.filter((r) => r !== rotulo) }
  if (!livres.has(rotulo)) return { rotulos }
  if (rotulos.length >= limite) return { rotulos, aviso: 'limite' }
  return { rotulos: [...rotulos, rotulo] }
}

export interface Pedido {
  itens: number
  precoCentavos: number
  taxaCentavos: number
  totalCentavos: number
}

export function calcularPedido(precoCentavos: number, rotulos: string[], taxaCentavos = 0): Pedido {
  const itens = rotulos.length
  return { itens, precoCentavos, taxaCentavos, totalCentavos: itens === 0 ? 0 : precoCentavos * itens + taxaCentavos }
}
