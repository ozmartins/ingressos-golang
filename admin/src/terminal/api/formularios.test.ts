import { describe, expect, it } from 'vitest'
import { FORMULARIOS, converter, montarCorpo, type CampoDef } from './formularios'

const c = (extra: Partial<CampoDef> & { tipo: CampoDef['tipo'] }): CampoDef =>
  ({ chave: 'k', rotulo: 'K', obrigatorio: true, ...extra }) as CampoDef

describe('converter', () => {
  it('texto: obrigatório, opcional e tamanho máximo', () => {
    expect(converter(c({ tipo: 'texto' }), '  ')).toEqual({ erro: 'campo obrigatório' })
    expect(converter(c({ tipo: 'texto', obrigatorio: false }), '')).toEqual({ valor: undefined })
    expect(converter(c({ tipo: 'texto', maxTamanho: 3 }), 'abcd')).toHaveProperty('erro')
    expect(converter(c({ tipo: 'texto' }), ' oi ')).toEqual({ valor: 'oi' })
  })

  it('texto: maiúsculas e formato', () => {
    const campo = c({ tipo: 'texto', maiusculas: true, formato: /^[A-Z]{2}$/ })
    expect(converter(campo, 'sc')).toEqual({ valor: 'SC' })
    expect(converter(campo, 's1')).toHaveProperty('erro')
  })

  it('inteiro: mínimo e não numérico', () => {
    const campo = c({ tipo: 'inteiro', minimo: 1 })
    expect(converter(campo, '90')).toEqual({ valor: 90 })
    expect(converter(campo, '0')).toEqual({ erro: 'o mínimo é 1' })
    expect(converter(campo, 'abc')).toEqual({ erro: 'informe um número inteiro' })
    expect(converter(campo, '1.5')).toHaveProperty('erro')
  })

  it('lógico aceita s/n, sim/não e true/false', () => {
    const campo = c({ tipo: 'logico', obrigatorio: false })
    for (const v of ['s', 'SIM', 'true']) expect(converter(campo, v)).toEqual({ valor: true })
    for (const v of ['n', 'não', 'nao', 'false']) expect(converter(campo, v)).toEqual({ valor: false })
    expect(converter(campo, 'talvez')).toEqual({ erro: 'responda s ou n' })
  })

  it('opção é case-insensitive e devolve o valor canônico', () => {
    const campo = c({ tipo: 'opcao', opcoes: ['EM_CARTAZ', 'BREVE'] })
    expect(converter(campo, 'breve')).toEqual({ valor: 'BREVE' })
    expect(converter(campo, 'x')).toEqual({ erro: 'opção inválida (use: EM_CARTAZ, BREVE)' })
  })

  it('uuid', () => {
    const campo = c({ tipo: 'uuid' })
    expect(converter(campo, '11111111-1111-1111-1111-111111111111')).toEqual({
      valor: '11111111-1111-1111-1111-111111111111',
    })
    expect(converter(campo, '123')).toHaveProperty('erro')
  })

  it('decimal: até 2 casas, devolvido como string', () => {
    const campo = c({ tipo: 'decimal' })
    expect(converter(campo, '42.50')).toEqual({ valor: '42.50' })
    expect(converter(campo, '42,5')).toEqual({ valor: '42.5' })
    expect(converter(campo, '10')).toEqual({ valor: '10' })
    expect(converter(campo, '1.234')).toHaveProperty('erro')
    expect(converter(campo, '-3')).toHaveProperty('erro')
  })

  it('datahora: local e ISO viram ISO com fuso; inválidas falham', () => {
    const campo = c({ tipo: 'datahora' })
    expect(converter(campo, '2026-12-01 19:30')).toEqual({
      valor: new Date(2026, 11, 1, 19, 30).toISOString(),
    })
    expect(converter(campo, '2026-12-01T19:30:00-03:00')).toEqual({ valor: '2026-12-01T22:30:00.000Z' })
    expect(converter(campo, '2026-02-31 10:00')).toHaveProperty('erro')
    expect(converter(campo, 'amanhã')).toHaveProperty('erro')
  })
})

describe('montarCorpo', () => {
  it('omite os campos não respondidos e preserva os tipos', () => {
    const campos: CampoDef[] = [c({ tipo: 'texto', chave: 'a' }), c({ tipo: 'inteiro', chave: 'b' }), c({ tipo: 'logico', chave: 'd' })]
    expect(montarCorpo(campos, { a: 'x', b: 2, d: false, extra: 1 })).toEqual({ a: 'x', b: 2, d: false })
    expect(montarCorpo(campos, { a: 'x', b: undefined })).toEqual({ a: 'x' })
  })
})

const chaves = (f: string) => FORMULARIOS[f].map((x) => `${x.chave}${x.obrigatorio ? '*' : ''}`)

describe('FORMULARIOS', () => {
  it('filmes criar/atualizar seguem FilmeEntrada', () => {
    const esperado = ['titulo*', 'sinopse', 'duracao_minutos*', 'classificacao_etaria*', 'genero*', 'imagem_url', 'status']
    expect(chaves('filmes.criar')).toEqual(esperado)
    expect(chaves('filmes.atualizar')).toEqual(esperado)
  })

  it('cinemas criar/atualizar seguem CinemaEntrada', () => {
    const esperado = ['nome*', 'cidade*', 'estado*', 'endereco*', 'ativo']
    expect(chaves('cinemas.criar')).toEqual(esperado)
    expect(chaves('cinemas.atualizar')).toEqual(esperado)
  })

  it('salas: criar tem cinema e fileiras; atualizar não', () => {
    expect(chaves('salas.criar')).toEqual(['cinema_id*', 'numero*', 'tipo_tela*', 'fileiras*', 'ativo'])
    expect(chaves('salas.atualizar')).toEqual(['numero*', 'tipo_tela*', 'ativo'])
  })

  it('sessões: atualizar não pede sala_id; preço é decimal (string)', () => {
    expect(chaves('sessoes.criar')).toEqual(['filme_id*', 'sala_id*', 'data_hora_inicio*', 'idioma*', 'preco_base*', 'status'])
    expect(chaves('sessoes.atualizar')).toEqual(['filme_id*', 'data_hora_inicio*', 'idioma*', 'preco_base*', 'status'])
    expect(FORMULARIOS['sessoes.criar'].find((x) => x.chave === 'preco_base')?.tipo).toBe('decimal')
  })

  it('padrões e opções do contrato', () => {
    const status = FORMULARIOS['filmes.criar'].find((x) => x.chave === 'status')
    expect(status).toMatchObject({ valorPadrao: 'EM_CARTAZ', opcoes: ['EM_CARTAZ', 'BREVE', 'FORA_DE_CARTAZ'] })
  })
})
