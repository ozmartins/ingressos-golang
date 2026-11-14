import { describe, expect, it } from 'vitest'
import type { Cinema } from '../api/tipos'
import { ajustarCidade, filtrarCinemas, filtrosCinemasVazios, opcoesCinemas } from './filtrosCinemas'

function cinema(id: string, nome: string, cidade: string, estado: string): Cinema {
  return { id, nome, cidade, estado, endereco: 'Rua X, 1', ativo: true }
}

const cinemas: Cinema[] = [
  cinema('1', 'Cine Centro', 'Florianópolis', 'SC'),
  cinema('2', 'Cine Beira-Mar', 'Florianópolis', 'SC'),
  cinema('3', 'Cine Vale', 'Joinville', 'SC'),
  cinema('4', 'Cine Paulista', 'São Paulo', 'SP'),
]

const ids = (lista: Cinema[]) => lista.map((c) => c.id)

describe('filtrarCinemas', () => {
  it('sem filtros devolve tudo', () => {
    expect(ids(filtrarCinemas(cinemas, filtrosCinemasVazios))).toEqual(['1', '2', '3', '4'])
  })

  it('busca no nome ignorando acento e caixa', () => {
    expect(ids(filtrarCinemas(cinemas, { ...filtrosCinemasVazios, texto: 'BEIRA' }))).toEqual(['2'])
    expect(ids(filtrarCinemas(cinemas, { ...filtrosCinemasVazios, texto: 'paulista' }))).toEqual(['4'])
  })

  it('filtra por estado e por cidade', () => {
    expect(ids(filtrarCinemas(cinemas, { ...filtrosCinemasVazios, estado: 'SC' }))).toEqual(['1', '2', '3'])
    expect(ids(filtrarCinemas(cinemas, { texto: '', estado: 'SC', cidade: 'Joinville' }))).toEqual(['3'])
  })

  it('combina critérios com E lógico', () => {
    expect(ids(filtrarCinemas(cinemas, { texto: 'cine c', estado: 'SC', cidade: 'Florianópolis' }))).toEqual(['1'])
  })
})

describe('opcoesCinemas', () => {
  it('lista estados distintos e ordenados', () => {
    expect(opcoesCinemas(cinemas, '').estados).toEqual(['SC', 'SP'])
  })

  it('sem estado, oferece todas as cidades', () => {
    expect(opcoesCinemas(cinemas, '').cidades).toEqual(['Florianópolis', 'Joinville', 'São Paulo'])
  })

  it('com estado, oferece só as cidades dele', () => {
    expect(opcoesCinemas(cinemas, 'SC').cidades).toEqual(['Florianópolis', 'Joinville'])
    expect(opcoesCinemas(cinemas, 'SP').cidades).toEqual(['São Paulo'])
  })
})

describe('ajustarCidade', () => {
  it('zera a cidade que não pertence ao estado escolhido', () => {
    const ajustado = ajustarCidade(cinemas, { texto: '', estado: 'SP', cidade: 'Joinville' })
    expect(ajustado.cidade).toBe('')
  })

  it('mantém a cidade que pertence ao estado', () => {
    const ajustado = ajustarCidade(cinemas, { texto: '', estado: 'SC', cidade: 'Joinville' })
    expect(ajustado.cidade).toBe('Joinville')
  })
})
