import { describe, expect, it } from 'vitest'
import { caminhoCinema, caminhoCompra, caminhoPagamento, lerRota } from './rotas'

describe('rotas', () => {
  it('monta o caminho do cinema', () => {
    expect(caminhoCinema('abc')).toBe('/cinemas/abc')
  })

  it('reconhece /cinemas/{id}, com ou sem barra final', () => {
    expect(lerRota('/cinemas/abc')).toEqual({ tipo: 'cinema', id: 'abc' })
    expect(lerRota('/cinemas/abc/')).toEqual({ tipo: 'cinema', id: 'abc' })
  })

  it('monta e reconhece o caminho da compra', () => {
    expect(caminhoCompra('c 1', 'f1')).toBe('/cinemas/c%201/filmes/f1/comprar')
    expect(caminhoCompra('c1', 'f1', 's1')).toBe('/cinemas/c1/filmes/f1/comprar?sessao=s1')
    expect(lerRota('/cinemas/c%201/filmes/f1/comprar/')).toEqual({ tipo: 'compra', cinemaId: 'c 1', filmeId: 'f1' })
  })

  it('monta e reconhece o caminho do pagamento', () => {
    expect(caminhoPagamento('c 1', 'f1')).toBe('/cinemas/c%201/filmes/f1/pagamento')
    expect(lerRota('/cinemas/c%201/filmes/f1/pagamento/')).toEqual({ tipo: 'pagamento', cinemaId: 'c 1', filmeId: 'f1' })
    expect(lerRota('/cinemas/a/filmes/%E0%A4%A/pagamento')).toEqual({ tipo: 'inicio' })
  })

  it.each(['/', '', '/outra', '/cinemas', '/cinemas/a/b', '/cinemas/%E0%A4%A', '/cinemas/a/filmes/b', '/cinemas/a/filmes/b/outra', '/cinemas/a/filmes/%E0%A4%A/comprar'])(
    'trata %j como página inicial',
    (caminho) => {
      expect(lerRota(caminho)).toEqual({ tipo: 'inicio' })
    },
  )
})
