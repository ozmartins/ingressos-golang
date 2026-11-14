import { describe, expect, it } from 'vitest'
import { formatarDataHora, formatarDia, formatarDuracao, formatarHora, formatarPreco } from './formatar'

const semEspacoRigido = (texto: string) => texto.replace(/ /g, ' ')

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

describe('formatação pt-BR', () => {
  it('formata o preço em reais', () => {
    expect(semEspacoRigido(formatarPreco('42.00'))).toBe('R$ 42,00')
    expect(semEspacoRigido(formatarPreco('1234.5'))).toBe('R$ 1.234,50')
  })

  it('formata a hora local', () => {
    expect(formatarHora(new Date(2026, 10, 14, 19, 5))).toBe('19:05')
  })

  it('formata o dia local com dia da semana', () => {
    expect(formatarDia('2026-11-14')).toMatch(/14\/11/)
    expect(formatarDia('2026-11-14')).toMatch(/s[áa]b/i)
  })

  it('formata data e hora', () => {
    expect(formatarDataHora(new Date(2026, 10, 14, 19, 5))).toBe('14/11/2026 19:05')
  })
})
