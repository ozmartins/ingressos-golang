import { describe, expect, it } from 'vitest'
import { AVISO_SIMULACAO, codigoPedido, montarIngressos, nomeArquivoIngresso } from './ingressoPdf'
import type { ReservaPagamento } from './reservaPagamento'

const base: ReservaPagamento = {
  usuarioId: null,
  cinemaId: 'c1',
  filmeId: 'f1',
  sessaoId: 's1',
  filmeTitulo: 'Duna: Parte Dois',
  classificacao: '12',
  posterUrl: null,
  cinemaNome: 'Cine Centro',
  sala: 3,
  inicio: new Date(2026, 10, 15, 16, 0).toISOString(),
  rotulos: ['B4', 'A10', 'A2'],
  precoCentavos: 3000,
  taxaCentavos: 250,
  totalCentavos: 9250,
  expiraEm: '2026-11-15T21:10:00.000Z',
}

describe('codigoPedido', () => {
  it('tem o formato ING-XXXXXXXX', () => {
    expect(codigoPedido(base)).toMatch(/^ING-[0-9A-Z]{8}$/)
  })

  it('é estável para o mesmo pedido, mesmo com os assentos em outra ordem', () => {
    expect(codigoPedido(base)).toBe(codigoPedido({ ...base }))
    expect(codigoPedido(base)).toBe(codigoPedido({ ...base, rotulos: ['A2', 'A10', 'B4'] }))
  })

  it.each([
    ['assentos', { rotulos: ['A1'] }],
    ['sessão', { sessaoId: 's2' }],
    ['prazo da reserva', { expiraEm: '2026-11-15T22:10:00.000Z' }],
  ])('muda com outro(a) %s', (_nome, mudanca) => {
    expect(codigoPedido({ ...base, ...mudanca })).not.toBe(codigoPedido(base))
  })
})

describe('nomeArquivoIngresso', () => {
  it('identifica filme e sessão, sem acento nem caracteres inválidos', () => {
    expect(nomeArquivoIngresso(base)).toBe('ingresso-duna-parte-dois-15-11-1600.pdf')
    expect(nomeArquivoIngresso({ ...base, filmeTitulo: 'Ação & Emoção: O Retorno/2?' })).toBe('ingresso-acao-emocao-o-retorno-2-15-11-1600.pdf')
  })

  it('limita o título a 40 caracteres sem hífen sobrando', () => {
    const nome = nomeArquivoIngresso({ ...base, filmeTitulo: 'A'.repeat(39) + ' B' + 'C'.repeat(30) })
    expect(nome).toMatch(/^ingresso-a{39}-15-11-1600\.pdf$|^ingresso-a{39}-b[c]*-15-11-1600\.pdf$/)
    expect(nome.length).toBeLessThanOrEqual('ingresso-'.length + 40 + '-15-11-1600.pdf'.length)
  })

  it('título sem letras vira "filme"', () => {
    expect(nomeArquivoIngresso({ ...base, filmeTitulo: '???' })).toBe('ingresso-filme-15-11-1600.pdf')
  })
})

describe('montarIngressos', () => {
  it('gera um ingresso por assento, em ordem de fileira e número, com os dados da sessão', () => {
    const r = montarIngressos(base)
    expect(r.ingressos.map((i) => i.assento)).toEqual(['A2', 'A10', 'B4'])
    expect(r.ingressos[0]).toMatchObject({ filme: 'Duna: Parte Dois', classificacao: '12', cinema: 'Cine Centro', sala: 'Sala 3', horario: '16:00' })
    expect(r.ingressos[0].dia).toMatch(/15\/11/)
    expect(r.codigoPedido).toBe(codigoPedido(base))
    expect(r.nomeArquivo).toBe(nomeArquivoIngresso(base))
  })

  it('resumo: valores em reais, uma vez', () => {
    const { resumo } = montarIngressos(base)
    expect(resumo.quantidade).toBe(3)
    expect(resumo.ingressos.replace(/\s/g, ' ')).toBe('R$ 90,00')
    expect(resumo.taxa.replace(/\s/g, ' ')).toBe('R$ 2,50')
    expect(resumo.total.replace(/\s/g, ' ')).toBe('R$ 92,50')
  })

  it('traz o aviso de simulação', () => {
    expect(montarIngressos(base).aviso).toBe(AVISO_SIMULACAO)
    expect(AVISO_SIMULACAO).toMatch(/simulado/)
  })

  it('omite a classificação quando a reserva não a tem', () => {
    const { classificacao: _omitida, ...semClassificacao } = base
    const r = montarIngressos(semClassificacao)
    expect(r.ingressos[0]).not.toHaveProperty('classificacao')
  })

  it.each([1, 3, 10])('%i assento(s): um ingresso para cada', (n) => {
    const rotulos = Array.from({ length: n }, (_, i) => `A${i + 1}`)
    expect(montarIngressos({ ...base, rotulos }).ingressos).toHaveLength(n)
  })

  it('não carrega nenhum campo de cartão', () => {
    const chaves = (v: unknown): string[] =>
      Array.isArray(v) ? v.flatMap(chaves) : v && typeof v === 'object' ? Object.entries(v).flatMap(([k, x]) => [k, ...chaves(x)]) : []
    const todas = chaves(montarIngressos(base)).join(' ').toLowerCase()
    for (const proibido of ['cartao', 'numero', 'cvv', 'codigoseguranca', 'nomeimpresso']) expect(todas).not.toContain(proibido)
  })
})
