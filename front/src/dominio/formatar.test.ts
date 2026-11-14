import { describe, expect, it } from 'vitest'
import { formatarDuracao } from './formatar'

describe('formatarDuracao', () => {
  it('formata horas e minutos', () => {
    expect(formatarDuracao(125)).toBe('2h 05min')
    expect(formatarDuracao(90)).toBe('1h 30min')
    expect(formatarDuracao(60)).toBe('1h 00min')
  })
  it('formata menos de uma hora', () => {
    expect(formatarDuracao(45)).toBe('45min')
  })
})
