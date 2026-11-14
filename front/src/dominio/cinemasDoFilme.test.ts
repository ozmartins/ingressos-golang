import { describe, expect, it } from 'vitest'
import type { Cinema, SessaoNaGrade } from '../api/tipos'
import { cinemasComOFilme } from './cinemasDoFilme'

const agora = new Date('2026-11-14T12:00:00Z')

function cinema(id: string, nome: string, cidade: string, estado: string): Cinema {
  return { id, nome, cidade, estado, endereco: 'Rua A', ativo: true }
}

function sessao(id: string, cinemaId: string, inicio: string): SessaoNaGrade {
  return {
    id,
    filme_id: 'f1',
    filme_titulo: 'Filme',
    cinema_id: cinemaId,
    cinema_nome: 'x',
    sala_numero: 1,
    tipo_tela: '2D',
    data_hora_inicio: inicio,
    idioma: 'DUBLADO',
    preco_base: '30.00',
  }
}

const futura = '2026-11-15T20:00:00Z'
const passada = '2026-11-13T20:00:00Z'

describe('cinemasComOFilme', () => {
  const cinemas = [
    cinema('a', 'Cine A', 'Florianópolis', 'SC'),
    cinema('b', 'Cine B', 'Curitiba', 'PR'),
    cinema('c', 'Cine C', 'Blumenau', 'SC'),
  ]

  it('mantém só cinemas com sessão futura e conta as sessões', () => {
    const r = cinemasComOFilme([sessao('1', 'a', futura), sessao('2', 'a', futura), sessao('3', 'b', passada)], cinemas, agora)
    expect(r.map((x) => [x.cinema.id, x.quantidadeSessoes])).toEqual([['a', 2]])
  })

  it('sessão já iniciada (ou começando agora) não conta', () => {
    expect(cinemasComOFilme([sessao('1', 'a', agora.toISOString())], cinemas, agora)).toEqual([])
  })

  it('ignora sessões de cinemas ausentes da lista de ativos', () => {
    expect(cinemasComOFilme([sessao('1', 'inativo', futura)], cinemas, agora)).toEqual([])
  })

  it('sem sessões devolve lista vazia', () => {
    expect(cinemasComOFilme([], cinemas, agora)).toEqual([])
  })

  it('ordena por UF, cidade e nome', () => {
    const todas = ['a', 'b', 'c'].map((id) => sessao(id, id, futura))
    expect(cinemasComOFilme(todas, cinemas, agora).map((x) => x.cinema.id)).toEqual(['b', 'c', 'a'])
  })
})
