import { describe, expect, it } from 'vitest'
import { FalhaRequisicao, montarUrl, requisitar } from './cliente'
import { BASE_TESTE, fakeFetch, fetchPendente, fetchQueFalha, respostaJson } from './fakes'

const get = { metodo: 'GET', caminho: '/filmes', raiz: false, query: {}, token: null } as const

describe('montarUrl', () => {
  it('usa o prefixo /api/v1 e a query string', () => {
    expect(montarUrl(BASE_TESTE, { caminho: '/filmes', raiz: false, query: { status: 'BREVE', page: '2' } })).toBe(
      '/catalogo/api/v1/filmes?status=BREVE&page=2',
    )
  })

  it('health fica na raiz, fora do prefixo', () => {
    expect(montarUrl(BASE_TESTE, { caminho: '/health', raiz: true, query: {} })).toBe('/catalogo/health')
  })
})

describe('requisitar', () => {
  it('devolve status e corpo JSON', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, { itens: [] }))
    const r = await requisitar(fn, BASE_TESTE, get)
    expect(r).toEqual({ status: 200, corpo: { itens: [] }, location: undefined })
  })

  it('envia Authorization só quando há token', async () => {
    const a = fakeFetch(() => respostaJson(200, {}))
    await requisitar(a.fn, BASE_TESTE, get)
    expect(a.chamadas[0].init.headers).not.toHaveProperty('Authorization')

    const b = fakeFetch(() => respostaJson(200, {}))
    await requisitar(b.fn, BASE_TESTE, { ...get, token: 'abc' })
    expect(b.chamadas[0].init.headers).toHaveProperty('Authorization', 'Bearer abc')
  })

  it('envia o corpo como JSON com Content-Type', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, { id: '1' }, { Location: '/api/v1/filmes/1' }))
    const r = await requisitar(fn, BASE_TESTE, {
      metodo: 'POST', caminho: '/filmes', raiz: false, query: {}, corpo: { titulo: 'X' }, token: 't',
    })
    expect(chamadas[0].init.method).toBe('POST')
    expect(chamadas[0].init.body).toBe('{"titulo":"X"}')
    expect(chamadas[0].init.headers).toHaveProperty('Content-Type', 'application/json')
    expect(r.location).toBe('/api/v1/filmes/1')
  })

  it('204 sem corpo devolve corpo nulo', async () => {
    const { fn } = fakeFetch(() => new Response(null, { status: 204 }))
    const r = await requisitar(fn, BASE_TESTE, { ...get, metodo: 'DELETE', token: 't' })
    expect(r).toMatchObject({ status: 204, corpo: null })
  })

  it('corpo que não é JSON vira texto', async () => {
    const { fn } = fakeFetch(() => new Response('Method Not Allowed', { status: 405 }))
    const r = await requisitar(fn, BASE_TESTE, get)
    expect(r.corpo).toBe('Method Not Allowed')
  })

  it('erro de rede vira FalhaRequisicao(rede)', async () => {
    await expect(requisitar(fetchQueFalha(), BASE_TESTE, get)).rejects.toMatchObject({ tipo: 'rede' })
  })

  it('tempo esgotado vira FalhaRequisicao(timeout)', async () => {
    const promessa = requisitar(fetchPendente(), BASE_TESTE, get, 20)
    await expect(promessa).rejects.toBeInstanceOf(FalhaRequisicao)
    await expect(requisitar(fetchPendente(), BASE_TESTE, get, 20)).rejects.toMatchObject({ tipo: 'timeout' })
  })
})
