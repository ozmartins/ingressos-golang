import { describe, expect, it } from 'vitest'
import { cartaoRecusado, formatarCvv, formatarNumero, formatarValidade, validarCartao, type DadosCartao } from './cartao'

const agora = new Date(2026, 10, 15) // 15/11/2026
const ok: DadosCartao = { numero: '4111 1111 1111 1111', nome: 'José da Silva', validade: '12/30', cvv: '123' }

describe('máscaras', () => {
  it('agrupa o número de 4 em 4 e ignora não dígitos', () => {
    expect(formatarNumero('4111111111111111')).toBe('4111 1111 1111 1111')
    expect(formatarNumero('4111-1111 11ab')).toBe('4111 1111 11')
    expect(formatarNumero('1'.repeat(30))).toBe('1111 1111 1111 1111 111')
  })
  it('formata a validade como MM/AA', () => {
    expect(formatarValidade('1')).toBe('1')
    expect(formatarValidade('123')).toBe('12/3')
    expect(formatarValidade('12/30abc')).toBe('12/30')
  })
  it('limita o CVV a 4 dígitos', () => {
    expect(formatarCvv('12a345')).toBe('1234')
  })
})

describe('validarCartao', () => {
  it('não acusa erro com dados válidos (inclusive colados com hífens)', () => {
    expect(validarCartao(ok, agora)).toEqual({})
    expect(validarCartao({ ...ok, numero: '4111-1111-1111-1111' }, agora)).toEqual({})
  })
  it('exige todos os campos', () => {
    expect(Object.keys(validarCartao({ numero: '', nome: ' ', validade: '', cvv: '' }, agora)).sort()).toEqual(['cvv', 'nome', 'numero', 'validade'])
  })
  it('aceita qualquer número preenchido (sem tamanho mínimo nem dígito verificador)', () => {
    for (const numero of ['1', '1234', '4111 1111 1111 1112', '0000 0000 0000 0000']) {
      expect(validarCartao({ ...ok, numero }, agora).numero).toBeUndefined()
    }
  })
  it('valida a validade: formato, mês, vencida e fim do mês', () => {
    expect(validarCartao({ ...ok, validade: '1230' }, agora).validade).toMatch(/MM\/AA/)
    expect(validarCartao({ ...ok, validade: '13/30' }, agora).validade).toMatch(/MM\/AA/)
    expect(validarCartao({ ...ok, validade: '00/30' }, agora).validade).toMatch(/MM\/AA/)
    expect(validarCartao({ ...ok, validade: '10/26' }, agora).validade).toBe('Cartão vencido.')
    expect(validarCartao({ ...ok, validade: '11/26' }, agora).validade).toBeUndefined()
  })
  it('valida o tamanho do CVV', () => {
    expect(validarCartao({ ...ok, cvv: '12' }, agora).cvv).toMatch(/inválido/)
    expect(validarCartao({ ...ok, cvv: '1234' }, agora).cvv).toBeUndefined()
  })
  it('aceita nome com acentos', () => {
    expect(validarCartao({ ...ok, nome: 'Conceição Ávila' }, agora).nome).toBeUndefined()
  })
})

describe('cartaoRecusado', () => {
  it('recusa números terminados em 0000', () => {
    expect(cartaoRecusado('4111 1110 0003 0000')).toBe(true)
    expect(cartaoRecusado('4111 1111 1111 1111')).toBe(false)
  })
})
