import { formatarCentavos, formatarDiaDe, formatarHora } from './formatar'
import type { ReservaPagamento } from './reservaPagamento'

export const AVISO_SIMULACAO = 'Pagamento simulado: este documento não tem validade para entrada.'

export interface IngressoDoAssento {
  assento: string
  filme: string
  classificacao?: string
  cinema: string
  sala: string
  dia: string
  horario: string
}

export interface IngressoPdf {
  codigoPedido: string
  nomeArquivo: string
  aviso: string
  resumo: { quantidade: number; ingressos: string; taxa: string; total: string }
  /** Um por assento, em ordem de fileira e número. */
  ingressos: IngressoDoAssento[]
}

/** "A10" → fileira "A", número 10; rótulos fora do padrão ficam por último, em ordem alfabética. */
function compararAssentos(a: string, b: string): number {
  const ma = /^([A-Za-z]+)(\d+)$/.exec(a)
  const mb = /^([A-Za-z]+)(\d+)$/.exec(b)
  if (ma && mb) return ma[1].localeCompare(mb[1]) || Number(ma[2]) - Number(mb[2])
  if (ma) return -1
  if (mb) return 1
  return a.localeCompare(b)
}

function fnv1a(texto: string, semente: number): number {
  let h = semente >>> 0
  for (let i = 0; i < texto.length; i++) {
    h ^= texto.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h
}

/** Código estável do pedido (mesma reserva → mesmo código): "ING-" + 8 caracteres. Não é prova de validade. */
export function codigoPedido(reserva: ReservaPagamento): string {
  const base = `${reserva.sessaoId}|${[...reserva.rotulos].sort(compararAssentos).join(',')}|${reserva.expiraEm}`
  const a = fnv1a(base, 0x811c9dc5).toString(36)
  const b = fnv1a(base, 0x9747b28c).toString(36)
  return `ING-${(a + b).padEnd(8, '0').slice(0, 8).toUpperCase()}`
}

const dois = (n: number) => String(n).padStart(2, '0')

/** "ingresso-duna-15-11-1600.pdf": título sem acentos, só [a-z0-9-], até 40 caracteres. */
export function nomeArquivoIngresso(reserva: ReservaPagamento): string {
  const titulo =
    reserva.filmeTitulo
      .normalize('NFD')
      .replace(/\p{Diacritic}/gu, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 40)
      .replace(/-+$/g, '') || 'filme'
  const inicio = new Date(reserva.inicio)
  return `ingresso-${titulo}-${dois(inicio.getDate())}-${dois(inicio.getMonth() + 1)}-${dois(inicio.getHours())}${dois(inicio.getMinutes())}.pdf`
}

/** Conteúdo do documento já formatado: um ingresso por assento, resumo de valores, código e aviso. Sem dado de cartão. */
export function montarIngressos(reserva: ReservaPagamento): IngressoPdf {
  const inicio = new Date(reserva.inicio)
  const comum = {
    filme: reserva.filmeTitulo,
    ...(reserva.classificacao ? { classificacao: reserva.classificacao } : {}),
    cinema: reserva.cinemaNome,
    sala: `Sala ${reserva.sala}`,
    dia: formatarDiaDe(inicio),
    horario: formatarHora(inicio),
  }
  return {
    codigoPedido: codigoPedido(reserva),
    nomeArquivo: nomeArquivoIngresso(reserva),
    aviso: AVISO_SIMULACAO,
    resumo: {
      quantidade: reserva.rotulos.length,
      ingressos: formatarCentavos(reserva.precoCentavos * reserva.rotulos.length),
      taxa: formatarCentavos(reserva.taxaCentavos),
      total: formatarCentavos(reserva.totalCentavos),
    },
    ingressos: [...reserva.rotulos].sort(compararAssentos).map((assento) => ({ assento, ...comum })),
  }
}
