import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  buscarCinema,
  buscarCinemas,
  buscarFilmes,
  buscarSalas,
  buscarSessoes,
  buscarTodosFilmes,
  CinemaNaoEncontrado,
} from './catalogo'
import type { Cinema, Filme, PaginaApi } from './tipos'

function filme(id: string, status: Filme['status'] = 'EM_CARTAZ'): Filme {
  return { id, titulo: `Filme ${id}`, duracao_minutos: 100, classificacao_etaria: 'L', genero: 'Drama', status }
}

function cinema(id: string): Cinema {
  return { id, nome: `Cinema ${id}`, cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A', ativo: true }
}

function pagina<T>(itens: T[], numero: number, temProxima: boolean): PaginaApi<T> {
  return { itens, pagina: { pagina: numero, tamanho: 100, total: itens.length, tem_proxima: temProxima } }
}

function resposta(corpo: unknown, status = 200): Response {
  return new Response(JSON.stringify(corpo), { status, headers: { 'Content-Type': 'application/json' } })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('buscarFilmes', () => {
  it('percorre as páginas enquanto houver próxima e descarta filmes fora de cartaz', async () => {
    const fetchSimulado = vi
      .fn()
      .mockResolvedValueOnce(resposta(pagina([filme('1'), filme('2', 'FORA_DE_CARTAZ')], 1, true)))
      .mockResolvedValueOnce(resposta(pagina([filme('3', 'BREVE')], 2, false)))
    vi.stubGlobal('fetch', fetchSimulado)

    const filmes = await buscarFilmes()

    expect(filmes.map((f) => f.id)).toEqual(['1', '3'])
    expect(fetchSimulado).toHaveBeenCalledTimes(2)
    const primeira = String(fetchSimulado.mock.calls[0][0])
    const segunda = String(fetchSimulado.mock.calls[1][0])
    expect(primeira).toContain('/api/v1/filmes')
    expect(primeira).toContain('page_size=100')
    expect(primeira).toContain('page=1')
    expect(segunda).toContain('page=2')
  })

  it('falha com status diferente de 200', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(resposta({}, 500)))
    await expect(buscarFilmes()).rejects.toThrow()
  })

  it('falha em erro de rede', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('rede')))
    await expect(buscarFilmes()).rejects.toThrow()
  })

  it('falha quando o JSON é inesperado', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(resposta({ qualquer: 'coisa' })))
    await expect(buscarFilmes()).rejects.toThrow()
  })
})

describe('buscarCinemas', () => {
  it('pede apenas cinemas ativos', async () => {
    const fetchSimulado = vi.fn().mockResolvedValue(resposta(pagina([cinema('1')], 1, false)))
    vi.stubGlobal('fetch', fetchSimulado)

    const cinemas = await buscarCinemas()

    expect(cinemas).toHaveLength(1)
    const url = String(fetchSimulado.mock.calls[0][0])
    expect(url).toContain('/api/v1/cinemas')
    expect(url).toContain('ativo=true')
  })
})

describe('detalhe do cinema', () => {
  it('buscarCinema devolve o cinema', async () => {
    const fetchSimulado = vi.fn().mockResolvedValue(resposta(cinema('9')))
    vi.stubGlobal('fetch', fetchSimulado)
    expect((await buscarCinema('9')).id).toBe('9')
    expect(String(fetchSimulado.mock.calls[0][0])).toContain('/api/v1/cinemas/9')
  })

  it.each([404, 400])('buscarCinema com %i lança CinemaNaoEncontrado', async (status) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(resposta({}, status)))
    await expect(buscarCinema('x')).rejects.toBeInstanceOf(CinemaNaoEncontrado)
  })

  it('buscarCinema com outro status lança erro comum', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(resposta({}, 500)))
    const erro = await buscarCinema('x').catch((e: unknown) => e)
    expect(erro).toBeInstanceOf(Error)
    expect(erro).not.toBeInstanceOf(CinemaNaoEncontrado)
  })

  it('buscarSalas filtra por cinema ativo e percorre páginas', async () => {
    const fetchSimulado = vi
      .fn()
      .mockResolvedValueOnce(resposta(pagina([{ id: 's1' }], 1, true)))
      .mockResolvedValueOnce(resposta(pagina([{ id: 's2' }], 2, false)))
    vi.stubGlobal('fetch', fetchSimulado)
    const salas = await buscarSalas('c1')
    expect(salas).toHaveLength(2)
    const url = String(fetchSimulado.mock.calls[0][0])
    expect(url).toContain('/api/v1/salas')
    expect(url).toContain('cinema_id=c1')
    expect(url).toContain('ativo=true')
  })

  it('buscarSessoes filtra por cinema e percorre páginas', async () => {
    const fetchSimulado = vi
      .fn()
      .mockResolvedValueOnce(resposta(pagina([{ id: 'a' }], 1, true)))
      .mockResolvedValueOnce(resposta(pagina([{ id: 'b' }], 2, false)))
    vi.stubGlobal('fetch', fetchSimulado)
    expect(await buscarSessoes('c1')).toHaveLength(2)
    const url = String(fetchSimulado.mock.calls[0][0])
    expect(url).toContain('/api/v1/sessoes')
    expect(url).toContain('cinema_id=c1')
  })

  it('buscarTodosFilmes mantém os fora de cartaz', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(resposta(pagina([filme('1', 'FORA_DE_CARTAZ')], 1, false))))
    expect((await buscarTodosFilmes()).map((f) => f.id)).toEqual(['1'])
  })
})
