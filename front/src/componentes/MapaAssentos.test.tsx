import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Poltrona } from '../api/tipos'
import { MapaAssentos } from './MapaAssentos'

const p = (fileira: string, numero: number, status: Poltrona['status'] = 'LIVRE', tipo: Poltrona['tipo'] = 'NORMAL'): Poltrona => ({
  rotulo: `${fileira}${numero}`,
  fileira,
  numero,
  tipo,
  status,
})

const poltronas = [p('A', 1), p('A', 2, 'OCUPADA'), p('B', 1, 'LIVRE', 'PCD'), p('B', 2, 'LIVRE', 'NAMORADEIRA')]

describe('MapaAssentos', () => {
  it('mostra fileiras, tela, legenda e nomes acessíveis', () => {
    render(<MapaAssentos poltronas={poltronas} selecionados={['A1']} onAlternar={() => undefined} />)
    expect(screen.getByRole('group', { name: 'Fileira A' })).toBeInTheDocument()
    expect(screen.getByText('TELA')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fileira A, assento 1, selecionado' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Fileira A, assento 2, ocupado' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Fileira B, assento 1, livre, acessível' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Fileira B, assento 2, livre, namoradeira' })).toBeEnabled()
    expect(screen.getByRole('list', { name: 'Legenda' })).toBeInTheDocument()
  })

  it('aciona onAlternar só em assentos livres/selecionados (inclusive por teclado)', async () => {
    const aoAlternar = vi.fn()
    render(<MapaAssentos poltronas={poltronas} selecionados={[]} onAlternar={aoAlternar} />)
    await userEvent.click(screen.getByRole('button', { name: /assento 2, ocupado/ }))
    expect(aoAlternar).not.toHaveBeenCalled()
    screen.getByRole('button', { name: /Fileira A, assento 1/ }).focus()
    await userEvent.keyboard('{Enter}')
    expect(aoAlternar).toHaveBeenCalledWith('A1')
  })

  it('exibe os números dos assentos ao ativar a opção', async () => {
    render(<MapaAssentos poltronas={[p('A', 7)]} selecionados={[]} onAlternar={() => undefined} />)
    const assento = screen.getByRole('button', { name: /assento 7/ })
    expect(assento).toHaveTextContent('')
    await userEvent.click(screen.getByRole('switch', { name: 'Exibir números dos assentos' }))
    expect(assento).toHaveTextContent('7')
  })

  it('desabilita tudo quando pedido', () => {
    render(<MapaAssentos poltronas={poltronas} selecionados={[]} onAlternar={() => undefined} desabilitado />)
    expect(screen.getByRole('button', { name: /Fileira A, assento 1/ })).toBeDisabled()
  })
})
