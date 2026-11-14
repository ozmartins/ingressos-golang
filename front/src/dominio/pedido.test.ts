import { describe, expect, it } from 'vitest'
import { alternarAssento, calcularPedido, LIMITE_ASSENTOS, precoEmCentavos } from './pedido'

describe('precoEmCentavos', () => {
  it.each([
    ['42.00', 4200],
    ['42', 4200],
    ['42.5', 4250],
    ['0.07', 7],
    ['19.99', 1999],
  ])('%s → %i', (entrada, esperado) => {
    expect(precoEmCentavos(entrada)).toBe(esperado)
  })
})

describe('alternarAssento', () => {
  const livres = new Set(['A1', 'A2', 'A3'])

  it('marca e desmarca', () => {
    const marcado = alternarAssento([], 'A1', livres)
    expect(marcado.rotulos).toEqual(['A1'])
    expect(alternarAssento(marcado.rotulos, 'A1', livres).rotulos).toEqual([])
  })

  it('ignora assento que não está livre', () => {
    expect(alternarAssento(['A1'], 'B9', livres)).toEqual({ rotulos: ['A1'] })
  })

  it('avisa e mantém a seleção ao passar do limite', () => {
    const todos = new Set(Array.from({ length: 12 }, (_, i) => `A${i + 1}`))
    const cheio = Array.from({ length: LIMITE_ASSENTOS }, (_, i) => `A${i + 1}`)
    expect(alternarAssento(cheio, 'A11', todos)).toEqual({ rotulos: cheio, aviso: 'limite' })
    expect(alternarAssento(cheio, 'A1', todos).rotulos).toHaveLength(LIMITE_ASSENTOS - 1)
  })
})

describe('calcularPedido', () => {
  it('sem itens, total zero', () => {
    expect(calcularPedido(4200, [])).toEqual({ itens: 0, precoCentavos: 4200, taxaCentavos: 0, totalCentavos: 0 })
  })

  it('preço × itens + taxa', () => {
    expect(calcularPedido(1999, ['A1', 'A2', 'A3']).totalCentavos).toBe(5997)
    expect(calcularPedido(1000, ['A1'], 250).totalCentavos).toBe(1250)
  })
})
