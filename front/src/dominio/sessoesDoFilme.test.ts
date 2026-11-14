import { describe, expect, it } from 'vitest'
import type { SessaoNaGrade } from '../api/tipos'
import { sessaoPadrao, sessoesDoFilme } from './sessoesDoFilme'

const agora = new Date(2026, 10, 14, 12, 0)

function s(id: string, filme: string, dia: number, hora: number): SessaoNaGrade {
  return {
    id,
    filme_id: filme,
    filme_titulo: filme,
    cinema_id: 'c1',
    cinema_nome: 'Cine',
    sala_numero: 1,
    tipo_tela: '2D',
    data_hora_inicio: new Date(2026, 10, dia, hora).toISOString(),
    idioma: 'DUBLADO',
    preco_base: '30.00',
  }
}

describe('sessoesDoFilme', () => {
  it('filtra pelo filme, descarta passadas e ordena', () => {
    const lista = sessoesDoFilme([s('c', 'f1', 15, 10), s('a', 'f1', 14, 18), s('x', 'f2', 14, 18), s('p', 'f1', 14, 9)], 'f1', agora)
    expect(lista.map((x) => x.id)).toEqual(['a', 'c'])
  })

  it('escolhe a sessão da URL ou cai na primeira', () => {
    const lista = [s('a', 'f1', 14, 18), s('b', 'f1', 15, 18)]
    expect(sessaoPadrao(lista, 'b')?.id).toBe('b')
    expect(sessaoPadrao(lista, 'zzz')?.id).toBe('a')
    expect(sessaoPadrao(lista, null)?.id).toBe('a')
    expect(sessaoPadrao([], 'a')).toBeUndefined()
  })
})
