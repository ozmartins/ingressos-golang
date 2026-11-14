import { describe, expect, it } from 'vitest'
import { caminhoCinema, lerRota } from './rotas'

describe('rotas', () => {
  it('monta o caminho do cinema', () => {
    expect(caminhoCinema('abc')).toBe('/cinemas/abc')
  })

  it('reconhece /cinemas/{id}, com ou sem barra final', () => {
    expect(lerRota('/cinemas/abc')).toEqual({ tipo: 'cinema', id: 'abc' })
    expect(lerRota('/cinemas/abc/')).toEqual({ tipo: 'cinema', id: 'abc' })
  })

  it.each(['/', '', '/outra', '/cinemas', '/cinemas/a/b', '/cinemas/%E0%A4%A'])(
    'trata %j como página inicial',
    (caminho) => {
      expect(lerRota(caminho)).toEqual({ tipo: 'inicio' })
    },
  )
})
