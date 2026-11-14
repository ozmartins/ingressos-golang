import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Cinema, Filme } from '../api/tipos'
import { PaginaInicial } from './PaginaInicial'

const filmes: Filme[] = [
  { id: '1', titulo: 'Aurora', duracao_minutos: 100, classificacao_etaria: 'L', genero: 'Drama', status: 'EM_CARTAZ' },
  { id: '2', titulo: 'Bravura', duracao_minutos: 110, classificacao_etaria: '12', genero: 'Ação', status: 'EM_CARTAZ' },
]
const cinemas: Cinema[] = [
  { id: '1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A', ativo: true },
  { id: '2', nome: 'Cine Vale', cidade: 'Joinville', estado: 'SC', endereco: 'Rua B', ativo: true },
]

function renderizar() {
  return render(
    <PaginaInicial
      carregarFilmes={() => Promise.resolve(filmes)}
      carregarCinemas={() => Promise.resolve(cinemas)}
    />,
  )
}

beforeEach(() => {
  window.history.replaceState(null, '', '/')
})

describe('PaginaInicial', () => {
  it('abre na aba Filmes, com Filmes antes de Cinemas', async () => {
    renderizar()
    const abas = screen.getAllByRole('tab')
    expect(abas.map((a) => a.textContent)).toEqual(['Filmes', 'Cinemas'])
    expect(abas[0]).toHaveAttribute('aria-selected', 'true')
    expect(await screen.findByText('Aurora')).toBeVisible()
  })

  it('mantém os filtros de filmes ao ir a cinemas e voltar', async () => {
    renderizar()
    await screen.findByText('Aurora')
    await userEvent.type(screen.getByLabelText('Buscar por título'), 'bravura')
    await userEvent.click(screen.getByRole('tab', { name: 'Cinemas' }))
    expect(await screen.findByText('Cine Centro')).toBeVisible()
    await userEvent.click(screen.getByRole('tab', { name: 'Filmes' }))
    expect(screen.getByLabelText('Buscar por título')).toHaveValue('bravura')
    expect(screen.getByText('Bravura')).toBeVisible()
    expect(screen.queryByText('Aurora')).not.toBeInTheDocument()
  })

  it('filtrar cinemas não altera a lista de filmes', async () => {
    renderizar()
    await screen.findByText('Aurora')
    await userEvent.click(screen.getByRole('tab', { name: 'Cinemas' }))
    await userEvent.type(await screen.findByLabelText('Buscar por nome'), 'vale')
    await userEvent.click(screen.getByRole('tab', { name: 'Filmes' }))
    expect(screen.getByText('Aurora')).toBeVisible()
    expect(screen.getByText('Bravura')).toBeVisible()
  })

  it('reflete a aba na URL e abre em cinemas quando pedido', async () => {
    window.history.replaceState(null, '', '/?aba=cinemas')
    renderizar()
    expect(screen.getByRole('tab', { name: 'Cinemas' })).toHaveAttribute('aria-selected', 'true')
    expect(await screen.findByText('Cine Vale')).toBeVisible()
    await userEvent.click(screen.getByRole('tab', { name: 'Filmes' }))
    expect(window.location.search).toBe('?aba=filmes')
  })

  it('troca de aba pelo teclado', async () => {
    renderizar()
    await screen.findByText('Aurora')
    screen.getByRole('tab', { name: 'Filmes' }).focus()
    await userEvent.keyboard('{ArrowRight}')
    expect(screen.getByRole('tab', { name: 'Cinemas' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Cinemas' })).toHaveFocus()
    await userEvent.keyboard('{Home}')
    expect(screen.getByRole('tab', { name: 'Filmes' })).toHaveAttribute('aria-selected', 'true')
    await userEvent.keyboard('{End}')
    expect(screen.getByRole('tab', { name: 'Cinemas' })).toHaveAttribute('aria-selected', 'true')
  })
})
