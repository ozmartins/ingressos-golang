import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import type { Cinema } from '../api/tipos'
import { filtrosCinemasVazios } from '../dominio/filtrosCinemas'
import { AbaCinemas } from './AbaCinemas'

const cinemas: Cinema[] = [
  { id: '1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A, 10', ativo: true },
  { id: '2', nome: 'Cine Vale', cidade: 'Joinville', estado: 'SC', endereco: 'Rua B, 20', ativo: true },
  { id: '3', nome: 'Cine Paulista', cidade: 'São Paulo', estado: 'SP', endereco: 'Av. C, 30', ativo: true },
]

function Aba() {
  const [filtros, setFiltros] = useState(filtrosCinemasVazios)
  return <AbaCinemas filtros={filtros} onFiltros={setFiltros} carregar={() => Promise.resolve(cinemas)} />
}

const nomes = () => screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)

describe('AbaCinemas', () => {
  it('renderiza os cinemas com cidade, estado e endereço', async () => {
    render(<Aba />)
    expect(await screen.findByText('Cine Centro')).toBeInTheDocument()
    expect(screen.getByText('Florianópolis / SC')).toBeInTheDocument()
    expect(screen.getByText('Rua A, 10')).toBeInTheDocument()
    expect(screen.getByText('3 cinemas')).toBeInTheDocument()
  })

  it('filtra por nome ao digitar', async () => {
    render(<Aba />)
    await screen.findByText('Cine Centro')
    await userEvent.type(screen.getByLabelText('Buscar por nome'), 'VALE')
    expect(nomes()).toEqual(['Cine Vale'])
  })

  it('limita as cidades ao estado escolhido e filtra por cidade', async () => {
    render(<Aba />)
    await screen.findByText('Cine Centro')
    await userEvent.selectOptions(screen.getByLabelText('Estado'), 'SC')
    expect(nomes()).toEqual(['Cine Centro', 'Cine Vale'])
    const cidades = within(screen.getByLabelText('Cidade')).getAllByRole('option').map((o) => o.textContent)
    expect(cidades).toEqual(['Todas as cidades', 'Florianópolis', 'Joinville'])
    await userEvent.selectOptions(screen.getByLabelText('Cidade'), 'Joinville')
    expect(nomes()).toEqual(['Cine Vale'])
  })

  it('zera a cidade ao trocar para um estado que não a contém', async () => {
    render(<Aba />)
    await screen.findByText('Cine Centro')
    await userEvent.selectOptions(screen.getByLabelText('Cidade'), 'Joinville')
    await userEvent.selectOptions(screen.getByLabelText('Estado'), 'SP')
    expect(nomes()).toEqual(['Cine Paulista'])
  })

  it('mostra sem resultado e limpa os filtros', async () => {
    render(<Aba />)
    await screen.findByText('Cine Centro')
    await userEvent.type(screen.getByLabelText('Buscar por nome'), 'zzz')
    expect(screen.getByText(/nenhum resultado/i)).toBeInTheDocument()
    const botoes = screen.getAllByRole('button', { name: 'Limpar filtros' })
    await userEvent.click(botoes[botoes.length - 1])
    expect(nomes()).toHaveLength(3)
  })
})
