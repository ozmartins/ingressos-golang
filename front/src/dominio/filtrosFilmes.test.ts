import { describe, expect, it } from 'vitest'
import type { Filme } from '../api/tipos'
import { filtrarFilmes, filtrosFilmesVazios, opcoesFilmes } from './filtrosFilmes'

function filme(parcial: Partial<Filme> & { id: string }): Filme {
  return {
    titulo: 'Título',
    duracao_minutos: 100,
    classificacao_etaria: 'L',
    genero: 'Drama',
    status: 'EM_CARTAZ',
    ...parcial,
  }
}

const filmes: Filme[] = [
  filme({ id: '1', titulo: 'A Invenção', genero: 'Ficção', classificacao_etaria: '12', status: 'EM_CARTAZ' }),
  filme({ id: '2', titulo: 'Ação Total', genero: 'Ação', classificacao_etaria: '16', status: 'BREVE' }),
  filme({ id: '3', titulo: 'Amor em Paris', genero: 'Romance', classificacao_etaria: 'L', status: 'EM_CARTAZ' }),
  filme({ id: '4', titulo: 'Ação Final', genero: 'Ação', classificacao_etaria: '12', status: 'EM_CARTAZ' }),
]

const ids = (lista: Filme[]) => lista.map((f) => f.id)

describe('filtrarFilmes', () => {
  it('sem filtros devolve tudo', () => {
    expect(ids(filtrarFilmes(filmes, filtrosFilmesVazios))).toEqual(['1', '2', '3', '4'])
  })

  it('busca no título ignorando acento e caixa', () => {
    expect(ids(filtrarFilmes(filmes, { ...filtrosFilmesVazios, texto: 'acao' }))).toEqual(['2', '4'])
    expect(ids(filtrarFilmes(filmes, { ...filtrosFilmesVazios, texto: '  INVENCAO ' }))).toEqual(['1'])
  })

  it('filtra por gênero, classificação e situação', () => {
    expect(ids(filtrarFilmes(filmes, { ...filtrosFilmesVazios, genero: 'Ação' }))).toEqual(['2', '4'])
    expect(ids(filtrarFilmes(filmes, { ...filtrosFilmesVazios, classificacao: '12' }))).toEqual(['1', '4'])
    expect(ids(filtrarFilmes(filmes, { ...filtrosFilmesVazios, status: 'BREVE' }))).toEqual(['2'])
  })

  it('trata gêneros múltiplos separados por vírgula', () => {
    const multiplos = [
      filme({ id: 'a', genero: 'Drama, Romance, Guerra' }),
      filme({ id: 'b', genero: 'Ação' }),
      filme({ id: 'c', genero: 'Romance' }),
    ]
    expect(ids(filtrarFilmes(multiplos, { ...filtrosFilmesVazios, genero: 'Romance' }))).toEqual(['a', 'c'])
    expect(opcoesFilmes(multiplos).generos).toEqual(['Ação', 'Drama', 'Guerra', 'Romance'])
  })

  it('combina critérios com E lógico', () => {
    const filtros = { texto: 'acao', genero: 'Ação', classificacao: '12', status: 'EM_CARTAZ' as const }
    expect(ids(filtrarFilmes(filmes, filtros))).toEqual(['4'])
  })

  it('devolve vazio quando nada atende', () => {
    expect(filtrarFilmes(filmes, { ...filtrosFilmesVazios, texto: 'zzz' })).toEqual([])
  })
})

describe('opcoesFilmes', () => {
  it('deriva opções distintas e ordenadas dos dados', () => {
    const opcoes = opcoesFilmes(filmes)
    expect(opcoes.generos).toEqual(['Ação', 'Ficção', 'Romance'])
    expect(opcoes.classificacoes).toEqual(['12', '16', 'L'])
  })
})
