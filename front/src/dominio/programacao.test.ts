import { describe, expect, it } from 'vitest'
import type { Filme, Sala, SessaoNaGrade } from '../api/tipos'
import {
  agruparPorFilme,
  diasDisponiveis,
  filtrarPorDia,
  montarSessoes,
  salasComSessoes,
} from './programacao'

function filme(id: string, titulo: string, status: Filme['status'] = 'EM_CARTAZ'): Filme {
  return { id, titulo, duracao_minutos: 100, classificacao_etaria: 'L', genero: 'Drama', status }
}

/** Início em horário local, para não depender do fuso da máquina. */
function sessao(id: string, filmeId: string, local: [number, number, number, number, number], sala = 1): SessaoNaGrade {
  const [a, m, d, h, min] = local
  return {
    id,
    filme_id: filmeId,
    filme_titulo: `t-${filmeId}`,
    cinema_id: 'c',
    cinema_nome: 'Cine',
    sala_numero: sala,
    tipo_tela: '2D',
    data_hora_inicio: new Date(a, m - 1, d, h, min).toISOString(),
    idioma: 'DUBLADO',
    preco_base: '42.00',
  }
}

const filmes = [filme('f1', 'Duna'), filme('f2', 'Avatar'), filme('f3', 'Futuro', 'BREVE'), filme('f4', 'Velho', 'FORA_DE_CARTAZ')]

describe('montarSessoes', () => {
  it('ordena por início (desempate pela sala), junta o filme e converte o preço', () => {
    const r = montarSessoes(
      [
        sessao('b', 'f1', [2026, 11, 14, 20, 0], 2),
        sessao('a', 'f2', [2026, 11, 14, 18, 0]),
        sessao('c', 'f1', [2026, 11, 14, 20, 0], 1),
      ],
      filmes,
    )
    expect(r.map((s) => s.id)).toEqual(['a', 'c', 'b'])
    expect(r[0].filme.titulo).toBe('Avatar')
    expect(r[0].preco).toBe(42)
    expect(r[0].dia).toBe('2026-11-14')
  })

  it('descarta sessões de filmes que não estão em cartaz ou que não constam da listagem', () => {
    const r = montarSessoes(
      [
        sessao('ok', 'f1', [2026, 11, 14, 18, 0]),
        sessao('breve', 'f3', [2026, 11, 14, 19, 0]),
        sessao('fora', 'f4', [2026, 11, 14, 20, 0]),
        sessao('desconhecido', 'zzz', [2026, 11, 14, 21, 0]),
      ],
      filmes,
    )
    expect(r.map((s) => s.id)).toEqual(['ok'])
  })
})

describe('salasComSessoes', () => {
  it('lista todas as salas por número, inclusive as sem sessões', () => {
    const salas = [
      { id: 's2', numero: 2 },
      { id: 's1', numero: 1 },
    ] as Sala[]
    const sessoes = montarSessoes([sessao('a', 'f1', [2026, 11, 14, 18, 0], 1), sessao('x', 'f3', [2026, 11, 14, 18, 0], 2)], filmes)
    expect(salasComSessoes(salas, sessoes).map((s) => [s.sala.id, s.quantidadeSessoes])).toEqual([
      ['s1', 1],
      ['s2', 0],
    ])
  })
})

describe('agruparPorFilme / dias / filtro', () => {
  const sessoes = montarSessoes(
    [
      sessao('1', 'f1', [2026, 11, 14, 20, 0]),
      sessao('2', 'f2', [2026, 11, 14, 18, 0]),
      sessao('3', 'f1', [2026, 11, 15, 16, 0]),
      sessao('4', 'f2', [2026, 11, 14, 22, 0]),
    ],
    filmes,
  )

  it('agrupa cada filme uma vez, com sessões em ordem, filmes pela primeira sessão', () => {
    const grupos = agruparPorFilme(sessoes)
    expect(grupos.map((g) => g.filme.titulo)).toEqual(['Avatar', 'Duna'])
    expect(grupos[0].sessoes.map((s) => s.id)).toEqual(['2', '4'])
    expect(grupos[1].sessoes.map((s) => s.id)).toEqual(['1', '3'])
  })

  it('desempata filmes com a mesma primeira sessão pelo título', () => {
    const iguais = montarSessoes(
      [sessao('1', 'f1', [2026, 11, 14, 18, 0]), sessao('2', 'f2', [2026, 11, 14, 18, 0], 2)],
      filmes,
    )
    expect(agruparPorFilme(iguais).map((g) => g.filme.titulo)).toEqual(['Avatar', 'Duna'])
  })

  it('lista os dias locais distintos em ordem', () => {
    expect(diasDisponiveis(sessoes)).toEqual(['2026-11-14', '2026-11-15'])
  })

  it('uma sessão às 22h locais pertence ao dia local', () => {
    expect(sessoes.find((s) => s.id === '4')?.dia).toBe('2026-11-14')
  })

  it('filtra por dia e remove filmes sem sessões nesse dia', () => {
    const doDia15 = filtrarPorDia(sessoes, '2026-11-15')
    expect(doDia15.map((s) => s.id)).toEqual(['3'])
    expect(agruparPorFilme(doDia15).map((g) => g.filme.titulo)).toEqual(['Duna'])
    expect(filtrarPorDia(sessoes, '')).toHaveLength(4)
  })
})
