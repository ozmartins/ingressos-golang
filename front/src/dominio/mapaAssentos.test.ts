import { describe, expect, it } from 'vitest'
import type { Poltrona } from '../api/tipos'
import { agruparPorFileira, rotuloAcessivel, rotulosLivres, situacaoDoAssento } from './mapaAssentos'

const p = (fileira: string, numero: number, status: Poltrona['status'] = 'LIVRE', tipo: Poltrona['tipo'] = 'NORMAL'): Poltrona => ({
  rotulo: `${fileira}${numero}`,
  fileira,
  numero,
  tipo,
  status,
})

describe('mapaAssentos', () => {
  it('agrupa por fileira e ordena', () => {
    const fileiras = agruparPorFileira([p('B', 2), p('A', 10), p('A', 2), p('B', 1)])
    expect(fileiras.map((f) => f.letra)).toEqual(['A', 'B'])
    expect(fileiras[0].poltronas.map((x) => x.rotulo)).toEqual(['A2', 'A10'])
  })

  it('lista só os rótulos livres', () => {
    expect([...rotulosLivres([p('A', 1), p('A', 2, 'OCUPADA'), p('A', 3, 'RESERVADA')])]).toEqual(['A1'])
  })

  it('deduz a situação do assento', () => {
    expect(situacaoDoAssento(p('A', 1), [])).toBe('livre')
    expect(situacaoDoAssento(p('A', 1), ['A1'])).toBe('selecionado')
    expect(situacaoDoAssento(p('A', 1, 'OCUPADA'), [])).toBe('ocupado')
    expect(situacaoDoAssento(p('A', 1, 'RESERVADA'), [])).toBe('ocupado')
    expect(situacaoDoAssento({ ...p('A', 1), status: 'OUTRO' as Poltrona['status'] }, [])).toBe('indisponivel')
  })

  it('monta o rótulo acessível', () => {
    expect(rotuloAcessivel(p('B', 4), 'livre')).toBe('Fileira B, assento 4, livre')
    expect(rotuloAcessivel(p('C', 1, 'LIVRE', 'PCD'), 'livre')).toBe('Fileira C, assento 1, livre, acessível')
    expect(rotuloAcessivel(p('C', 2, 'OCUPADA', 'NAMORADEIRA'), 'ocupado')).toBe('Fileira C, assento 2, ocupado, namoradeira')
  })
})
