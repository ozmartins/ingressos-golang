import { describe, expect, it } from 'vitest'
import { lerArgumentos, lerCorpo } from './argumentos'
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

describe('lerCorpo e lerArgumentos - corpo', () => {
  it('aceita JSON com espaços e aspas, sem escape', () => {
    const r = lerCorpo('{"titulo": "O Auto da Compadecida", "duracao_minutos": 104}')
    expect(r).toEqual({ corpo: { titulo: 'O Auto da Compadecida', duracao_minutos: 104 } })
  })

  it('corpo ausente vira erro de uso', () => {
    expect(lerCorpo('   ')).toEqual({ erro: 'corpo JSON ausente' })
    expect(lerArgumentos(def('filmes', 'criar'), '')).toEqual({ erro: 'corpo JSON ausente' })
  })

  it('JSON inválido vira erro de sintaxe', () => {
    const r = lerCorpo('{titulo')
    expect(r).toHaveProperty('erro')
    expect((r as { erro: string }).erro.startsWith('JSON inválido')).toBe(true)
  })

  it('criar lê o corpo inteiro; atualizar lê id e corpo', () => {
    expect(lerArgumentos(def('filmes', 'criar'), '{"a": 1}')).toEqual({ query: {}, corpo: { a: 1 } })
    expect(lerArgumentos(def('salas', 'atualizar'), 'id-1 {"numero": 2}')).toEqual({
      id: 'id-1',
      query: {},
      corpo: { numero: 2 },
    })
  })

  it('atualizar sem corpo ou sem id falha', () => {
    expect(lerArgumentos(def('salas', 'atualizar'), 'id-1')).toEqual({ erro: 'corpo JSON ausente' })
    expect(lerArgumentos(def('salas', 'atualizar'), '')).toEqual({ erro: 'identificador ausente' })
  })
})
