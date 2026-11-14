import { describe, expect, it } from 'vitest'
import { adicionar, anterior, criar, proximo } from './historico'

describe('historico', () => {
  it('adicionar ignora entradas vazias', () => {
    const h = adicionar(adicionar(criar(), ''), '   ')
    expect(h.itens).toEqual([])
  })

  it('anterior percorre do mais recente ao mais antigo e para no primeiro', () => {
    let h = adicionar(adicionar(criar(), 'a'), 'b')
    let r = anterior(h)
    expect(r.valor).toBe('b')
    r = anterior(r.historico)
    expect(r.valor).toBe('a')
    r = anterior(r.historico)
    expect(r.valor).toBe('a')
    h = r.historico
  })

  it('proximo avança e, passando do último, devolve vazio', () => {
    const h = adicionar(adicionar(criar(), 'a'), 'b')
    const r1 = anterior(anterior(h).historico)
    expect(r1.valor).toBe('a')
    const r2 = proximo(r1.historico)
    expect(r2.valor).toBe('b')
    const r3 = proximo(r2.historico)
    expect(r3.valor).toBe('')
    expect(proximo(r3.historico).valor).toBe('')
  })

  it('histórico vazio devolve vazio ao navegar', () => {
    expect(anterior(criar()).valor).toBe('')
    expect(proximo(criar()).valor).toBe('')
  })

  it('novo comando reinicia o ponteiro', () => {
    let h = adicionar(adicionar(criar(), 'a'), 'b')
    h = anterior(anterior(h).historico).historico
    h = adicionar(h, 'c')
    expect(anterior(h).valor).toBe('c')
  })
})
