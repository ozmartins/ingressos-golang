import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { SessaoNaGrade } from '../api/tipos'
import { CabecalhoSessao } from './CabecalhoSessao'

const s = (id: string, dia: number, sala: number): SessaoNaGrade => ({
  id,
  filme_id: 'f1',
  filme_titulo: 'F',
  cinema_id: 'c1',
  cinema_nome: 'Cine Centro',
  sala_numero: sala,
  tipo_tela: '2D',
  data_hora_inicio: new Date(2026, 10, dia, 19, 0).toISOString(),
  idioma: 'DUBLADO',
  preco_base: '30.00',
})

describe('CabecalhoSessao', () => {
  it('mostra etapa, cinema, sala e horário', () => {
    render(<CabecalhoSessao sessao={s('a', 14, 3)} sessoes={[s('a', 14, 3)]} onTrocar={() => undefined} />)
    expect(screen.getByText(/Selecione seus assentos/)).toBeInTheDocument()
    expect(screen.getByText('Cine Centro - Sala 3')).toBeInTheDocument()
    expect(screen.getByText('19:00')).toBeInTheDocument()
    expect(screen.queryByLabelText('Sessão')).not.toBeInTheDocument()
  })

  it('permite trocar de sessão quando há mais de uma', async () => {
    const aoTrocar = vi.fn()
    const lista = [s('a', 14, 3), s('b', 15, 2)]
    render(<CabecalhoSessao sessao={lista[0]} sessoes={lista} onTrocar={aoTrocar} />)
    await userEvent.selectOptions(screen.getByLabelText('Sessão'), 'b')
    expect(aoTrocar).toHaveBeenCalledWith('b')
  })
})
