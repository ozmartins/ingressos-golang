import { describe, expect, it } from 'vitest'
import { adicionarLinhas, MAX_LINHAS, type Linha } from './linhas'

const linha = (texto: string): Linha => ({ tipo: 'saida', texto })

describe('adicionarLinhas', () => {
  it('acrescenta as novas linhas ao final', () => {
    const r = adicionarLinhas([linha('a')], [linha('b'), linha('c')])
    expect(r.map((l) => l.texto)).toEqual(['a', 'b', 'c'])
  })

  it('não altera o array original', () => {
    const atuais = [linha('a')]
    adicionarLinhas(atuais, [linha('b')])
    expect(atuais).toHaveLength(1)
  })

  it('descarta as mais antigas ao passar do limite', () => {
    const atuais = Array.from({ length: MAX_LINHAS }, (_, i) => linha(String(i)))
    const r = adicionarLinhas(atuais, [linha('nova1'), linha('nova2')])
    expect(r).toHaveLength(MAX_LINHAS)
    expect(r[0].texto).toBe('2')
    expect(r[r.length - 1].texto).toBe('nova2')
  })
})
