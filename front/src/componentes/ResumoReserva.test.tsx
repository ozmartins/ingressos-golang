import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { ReservaPagamento } from '../dominio/reservaPagamento'
import { ResumoReserva } from './ResumoReserva'

const reserva: ReservaPagamento = {
  usuarioId: null,
  cinemaId: 'c1',
  filmeId: 'f1',
  sessaoId: 's1',
  filmeTitulo: 'Rocky',
  posterUrl: null,
  cinemaNome: 'Cine Centro',
  sala: 3,
  inicio: '2026-11-15T22:00:00.000Z',
  rotulos: ['A1', 'A2'],
  precoCentavos: 3000,
  taxaCentavos: 0,
  totalCentavos: 6000,
  expiraEm: '2026-11-15T21:10:00.000Z',
}

describe('ResumoReserva', () => {
  it('mostra filme, cinema, sala, assentos, itens, total e a contagem', () => {
    render(<ResumoReserva reserva={reserva} segundosRestantes={125} />)
    expect(screen.getByRole('heading', { name: 'Rocky' })).toBeInTheDocument()
    expect(screen.getByText('Cine Centro')).toBeInTheDocument()
    expect(screen.getByText('Sala 3')).toBeInTheDocument()
    expect(screen.getByText('Assentos: A1, A2')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText(/60,00/)).toBeInTheDocument()
    expect(screen.getByRole('timer')).toHaveTextContent('02:05')
  })

  it('usa o marcador padrão sem pôster e oculta a contagem quando null', () => {
    const { container } = render(<ResumoReserva reserva={reserva} segundosRestantes={null} />)
    expect(container.querySelector('img')).toBeNull()
    expect(screen.queryByRole('timer')).not.toBeInTheDocument()
  })

  it('mostra o pôster quando existe', () => {
    render(<ResumoReserva reserva={{ ...reserva, posterUrl: 'http://x/p.jpg' }} segundosRestantes={1} />)
    expect(screen.getByAltText('Pôster de Rocky')).toHaveAttribute('src', 'http://x/p.jpg')
  })
})
