import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Sala } from '../api/tipos'
import { ListaSalas } from './ListaSalas'

const sala = (numero: number): Sala => ({
  id: `s${numero}`,
  cinema_id: 'c',
  numero,
  tipo_tela: 'IMAX',
  capacidade_total: 120,
  ativo: true,
})

describe('ListaSalas', () => {
  it('mostra número, tipo de tela, capacidade e a situação das sessões', () => {
    render(<ListaSalas salas={[{ sala: sala(1), quantidadeSessoes: 3 }, { sala: sala(2), quantidadeSessoes: 0 }]} />)
    expect(screen.getByText('Sala 1')).toBeInTheDocument()
    expect(screen.getAllByText('IMAX · 120 lugares')).toHaveLength(2)
    expect(screen.getByText('3 sessões')).toBeInTheDocument()
    expect(screen.getByText('Sem sessões programadas')).toBeInTheDocument()
  })

  it('informa quando não há salas', () => {
    render(<ListaSalas salas={[]} />)
    expect(screen.getByText('Este cinema ainda não tem salas cadastradas.')).toBeInTheDocument()
  })
})
