import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  bloquearPoltronas,
  buscarMapa,
  ConflitoPoltronas,
  ErroEstoque,
  LimiteExcedido,
  NaoAutenticado,
  SessaoIndisponivel,
} from './estoque'

afterEach(() => vi.unstubAllGlobals())

function resposta(status: number, corpo: unknown = {}) {
  return Promise.resolve(new Response(JSON.stringify(corpo), { status }))
}

const poltrona = { rotulo: 'A1', fileira: 'A', numero: 1, tipo: 'NORMAL', status: 'LIVRE' }

describe('buscarMapa', () => {
  it('envia o token e devolve as poltronas', async () => {
    const fetchFn = vi.fn(() => resposta(200, { sessao_id: 's', poltronas: [poltrona] }))
    vi.stubGlobal('fetch', fetchFn)
    expect(await buscarMapa('s 1', 'tok')).toEqual([poltrona])
    const [url, init] = fetchFn.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/estoque/api/v1/sessoes/s%201/poltronas')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it.each([
    [401, NaoAutenticado],
    [404, SessaoIndisponivel],
    [422, SessaoIndisponivel],
    [503, ErroEstoque],
  ])('HTTP %i vira o erro tipado', async (status, tipo) => {
    vi.stubGlobal('fetch', () => resposta(status))
    await expect(buscarMapa('s', 't')).rejects.toBeInstanceOf(tipo)
  })

  it('rejeita corpo inesperado', async () => {
    vi.stubGlobal('fetch', () => resposta(200, { poltronas: [{}] }))
    await expect(buscarMapa('s', 't')).rejects.toThrow('inesperada')
  })
})

describe('bloquearPoltronas', () => {
  it('posta os rótulos e devolve a reserva', async () => {
    const fetchFn = vi.fn(() => resposta(201, { reserva_id: 'r', expira_em: '2026-11-14T20:00:00Z', mensagem: 'ok' }))
    vi.stubGlobal('fetch', fetchFn)
    const r = await bloquearPoltronas('s', ['A1', 'A2'], 'tok')
    expect(r.reserva_id).toBe('r')
    const [, init] = fetchFn.mock.calls[0] as unknown as [string, RequestInit]
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body as string)).toEqual({ poltronas_ids: ['A1', 'A2'] })
  })

  it('409 vira conflito e 401 vira não autenticado', async () => {
    vi.stubGlobal('fetch', () => resposta(409))
    await expect(bloquearPoltronas('s', ['A1'], 't')).rejects.toBeInstanceOf(ConflitoPoltronas)
    vi.stubGlobal('fetch', () => resposta(401))
    await expect(bloquearPoltronas('s', ['A1'], 't')).rejects.toBeInstanceOf(NaoAutenticado)
  })

  it('400 com metadata.limite vira LimiteExcedido', async () => {
    vi.stubGlobal('fetch', () => resposta(400, { type: 'x', title: 't', status: 400, metadata: { limite: '10' } }))
    await expect(bloquearPoltronas('s', ['A1'], 't')).rejects.toMatchObject({ limite: 10 })
    vi.stubGlobal('fetch', () => resposta(400, { type: 'x', title: 't', status: 400 }))
    await expect(bloquearPoltronas('s', ['A1'], 't')).rejects.toBeInstanceOf(ErroEstoque)
    vi.stubGlobal('fetch', () => resposta(400, { metadata: { limite: '10' } }))
    await expect(bloquearPoltronas('s', ['A1'], 't')).rejects.toBeInstanceOf(LimiteExcedido)
  })
})
