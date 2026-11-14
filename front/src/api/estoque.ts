import type { MapaPoltronas, Poltrona, Problem, RespostaBloqueio } from './tipos'

function base(): string {
  return (import.meta.env.VITE_ESTOQUE_URL ?? '/estoque').replace(/\/$/, '')
}

/** Token ausente, expirado ou recusado: é preciso entrar de novo. */
export class NaoAutenticado extends Error {
  constructor() {
    super('Sessão de login expirada.')
    this.name = 'NaoAutenticado'
  }
}

/** A sessão não aceita reserva ou não tem mapa de poltronas. */
export class SessaoIndisponivel extends Error {
  constructor() {
    super('A seleção de assentos não está disponível para esta sessão.')
    this.name = 'SessaoIndisponivel'
  }
}

/** Alguma poltrona pedida já está reservada ou ocupada. */
export class ConflitoPoltronas extends Error {
  constructor() {
    super('Alguns assentos escolhidos já não estão disponíveis.')
    this.name = 'ConflitoPoltronas'
  }
}

/** Mais poltronas do que o estoque aceita por pedido. */
export class LimiteExcedido extends Error {
  readonly limite: number | undefined
  constructor(limite: number | undefined) {
    super('Limite de assentos por pedido excedido.')
    this.limite = limite
    this.name = 'LimiteExcedido'
  }
}

export class ErroEstoque extends Error {
  readonly status: number
  constructor(status: number) {
    super(`Falha ao consultar o estoque (HTTP ${status}).`)
    this.status = status
    this.name = 'ErroEstoque'
  }
}

async function lerProblema(resposta: Response): Promise<Problem | undefined> {
  try {
    return (await resposta.json()) as Problem
  } catch {
    return undefined
  }
}

async function traduzirErro(resposta: Response): Promise<Error> {
  switch (resposta.status) {
    case 401:
      return new NaoAutenticado()
    case 404:
    case 422:
      return new SessaoIndisponivel()
    case 409:
      return new ConflitoPoltronas()
    case 400: {
      const problema = await lerProblema(resposta)
      const limite = Number(problema?.metadata?.limite)
      return Number.isFinite(limite) && problema?.metadata?.limite !== undefined
        ? new LimiteExcedido(limite)
        : new ErroEstoque(400)
    }
    default:
      return new ErroEstoque(resposta.status)
  }
}

function ehPoltrona(valor: unknown): valor is Poltrona {
  if (typeof valor !== 'object' || valor === null) return false
  const p = valor as Partial<Poltrona>
  return typeof p.rotulo === 'string' && typeof p.fileira === 'string' && typeof p.numero === 'number'
}

/** Mapa de poltronas da sessão (exige login). */
export async function buscarMapa(sessaoId: string, token: string): Promise<Poltrona[]> {
  const resposta = await fetch(`${base()}/api/v1/sessoes/${encodeURIComponent(sessaoId)}/poltronas`, {
    headers: { Accept: 'application/json', Authorization: `Bearer ${token}` },
  })
  if (!resposta.ok) throw await traduzirErro(resposta)
  const corpo = (await resposta.json()) as Partial<MapaPoltronas>
  if (!Array.isArray(corpo.poltronas) || !corpo.poltronas.every(ehPoltrona)) {
    throw new Error('Resposta inesperada do estoque.')
  }
  return corpo.poltronas
}

/** Reserva temporariamente as poltronas (rótulos) da sessão. */
export async function bloquearPoltronas(sessaoId: string, rotulos: string[], token: string): Promise<RespostaBloqueio> {
  const resposta = await fetch(`${base()}/api/v1/sessoes/${encodeURIComponent(sessaoId)}/bloqueios`, {
    method: 'POST',
    headers: { Accept: 'application/json', 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ poltronas_ids: rotulos }),
  })
  if (!resposta.ok) throw await traduzirErro(resposta)
  const corpo = (await resposta.json()) as Partial<RespostaBloqueio>
  if (typeof corpo.reserva_id !== 'string' || typeof corpo.expira_em !== 'string') {
    throw new Error('Resposta inesperada do estoque.')
  }
  return corpo as RespostaBloqueio
}
