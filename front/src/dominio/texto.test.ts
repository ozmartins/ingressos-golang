import { describe, expect, it } from 'vitest'
import { normalizar, ordenarPtBr } from './texto'

describe('normalizar', () => {
  it('ignora caixa e acentos', () => {
    expect(normalizar('São PAULO')).toBe('sao paulo')
    expect(normalizar('Ação')).toBe('acao')
  })

  it('remove espaços nas pontas', () => {
    expect(normalizar('  Matrix  ')).toBe('matrix')
  })

  it('devolve vazio para vazio', () => {
    expect(normalizar('')).toBe('')
    expect(normalizar('   ')).toBe('')
  })
})

describe('ordenarPtBr', () => {
  it('ordena respeitando acentos e remove duplicados', () => {
    expect(ordenarPtBr(['Drama', 'Ação', 'Comédia', 'Drama'])).toEqual(['Ação', 'Comédia', 'Drama'])
  })
})
