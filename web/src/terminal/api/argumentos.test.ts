import { describe, expect, it } from 'vitest'
import { lerArgumentos } from './argumentos'
import { buscarDefinicao } from './tabela'

const def = (c: string, a: string) => {
  const d = buscarDefinicao(c, a)
  if (!d) throw new Error('definição inexistente')
  return d
}

describe('lerArgumentos - listar', () => {
  it('aceita filtros e paginação', () => {
    expect(lerArgumentos(def('filmes', 'listar'), 'status=EM_CARTAZ page=2 page_size=5')).toEqual({
      query: { status: 'EM_CARTAZ', page: '2', page_size: '5' },
    })
  })

  it('aceita nenhum argumento', () => {
    expect(lerArgumentos(def('cinemas', 'listar'), '')).toEqual({ query: {} })
  })

  it('recusa filtro desconhecido', () => {
    expect(lerArgumentos(def('filmes', 'listar'), 'cor=azul')).toEqual({
      erro: 'filtro desconhecido: cor (filtros: status, page, page_size)',
    })
  })

  it('recusa argumento que não é chave=valor', () => {
    expect(lerArgumentos(def('filmes', 'listar'), 'abc')).toEqual({
      erro: 'argumento inválido: abc (use chave=valor)',
    })
    expect(lerArgumentos(def('filmes', 'listar'), 'status=')).toEqual({
      erro: 'argumento inválido: status= (use chave=valor)',
    })
  })
})

describe('lerArgumentos - id', () => {
  it('obter e remover exigem exatamente um id', () => {
    expect(lerArgumentos(def('filmes', 'obter'), ' abc ')).toEqual({ id: 'abc', query: {} })
    expect(lerArgumentos(def('filmes', 'remover'), 'abc')).toEqual({ id: 'abc', query: {} })
    expect(lerArgumentos(def('filmes', 'obter'), '')).toEqual({ erro: 'identificador ausente' })
    expect(lerArgumentos(def('filmes', 'obter'), 'a b')).toEqual({ erro: 'argumento inesperado: b' })
  })

  it('health não recebe argumentos', () => {
    expect(lerArgumentos(def('operacional', 'health'), '')).toEqual({ query: {} })
    expect(lerArgumentos(def('operacional', 'health'), 'x')).toEqual({ erro: 'health não recebe argumentos' })
  })
})

describe('lerArgumentos - escrita guiada', () => {
  it('criar não recebe argumentos', () => {
    expect(lerArgumentos(def('filmes', 'criar'), '')).toEqual({ query: {} })
    expect(lerArgumentos(def('filmes', 'criar'), '  ')).toEqual({ query: {} })
  })

  it('criar recusa JSON na linha e orienta o novo uso', () => {
    const r = lerArgumentos(def('filmes', 'criar'), '{"a": 1}')
    expect(r).toHaveProperty('erro')
    expect((r as { erro: string }).erro).toContain('digite apenas "filmes criar"')
  })

  it('atualizar recebe só o id', () => {
    expect(lerArgumentos(def('salas', 'atualizar'), 'id-1')).toEqual({ id: 'id-1', query: {} })
    expect(lerArgumentos(def('salas', 'atualizar'), '')).toEqual({ erro: 'identificador ausente' })
    const r = lerArgumentos(def('salas', 'atualizar'), 'id-1 {"numero": 2}')
    expect((r as { erro: string }).erro).toContain('digite apenas "salas atualizar <id>"')
  })
})
