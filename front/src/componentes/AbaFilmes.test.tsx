import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import type { Filme } from '../api/tipos'
import { filtrosFilmesVazios } from '../dominio/filtrosFilmes'
import { AbaFilmes } from './AbaFilmes'

const filmes: Filme[] = [
  { id: '1', titulo: 'A Invenção', duracao_minutos: 125, classificacao_etaria: '12', genero: 'Ficção', status: 'EM_CARTAZ' },
  { id: '2', titulo: 'Ação Total', duracao_minutos: 95, classificacao_etaria: '16', genero: 'Ação', status: 'BREVE' },
  { id: '3', titulo: 'Amor em Paris', duracao_minutos: 100, classificacao_etaria: 'L', genero: 'Romance', status: 'EM_CARTAZ' },
]

function Aba({ carregar }: { carregar: () => Promise<Filme[]> }) {
  const [filtros, setFiltros] = useState(filtrosFilmesVazios)
  return <AbaFilmes filtros={filtros} onFiltros={setFiltros} carregar={carregar} />
}

const titulos = () => screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)

describe('AbaFilmes', () => {
  it('renderiza os filmes com detalhes e contagem', async () => {
    render(<Aba carregar={() => Promise.resolve(filmes)} />)
    expect(await screen.findByText('A Invenção')).toBeInTheDocument()
    expect(screen.getByText(/2h 05min/)).toBeInTheDocument()
    expect(screen.getByText('3 filmes')).toBeInTheDocument()
  })

  it('filtra por título ao digitar', async () => {
    render(<Aba carregar={() => Promise.resolve(filmes)} />)
    await screen.findByText('A Invenção')
    await userEvent.type(screen.getByLabelText('Buscar por título'), 'acao')
    expect(titulos()).toEqual(['Ação Total'])
    expect(screen.getByText('1 filme')).toBeInTheDocument()
  })

  it('filtra por gênero, classificação e situação', async () => {
    render(<Aba carregar={() => Promise.resolve(filmes)} />)
    await screen.findByText('A Invenção')
    await userEvent.selectOptions(screen.getByLabelText('Gênero'), 'Romance')
    expect(titulos()).toEqual(['Amor em Paris'])
    await userEvent.click(screen.getByRole('button', { name: 'Limpar filtros' }))
    await userEvent.selectOptions(screen.getByLabelText('Situação'), 'Em breve')
    expect(titulos()).toEqual(['Ação Total'])
    await userEvent.click(screen.getByRole('button', { name: 'Limpar filtros' }))
    await userEvent.selectOptions(screen.getByLabelText('Classificação'), '12')
    expect(titulos()).toEqual(['A Invenção'])
  })

  it('oferece apenas gêneros existentes', async () => {
    render(<Aba carregar={() => Promise.resolve(filmes)} />)
    await screen.findByText('A Invenção')
    const opcoes = within(screen.getByLabelText('Gênero')).getAllByRole('option').map((o) => o.textContent)
    expect(opcoes).toEqual(['Todos os gêneros', 'Ação', 'Ficção', 'Romance'])
  })

  it('mostra sem resultado e limpa pelo atalho', async () => {
    render(<Aba carregar={() => Promise.resolve(filmes)} />)
    await screen.findByText('A Invenção')
    await userEvent.type(screen.getByLabelText('Buscar por título'), 'zzz')
    expect(screen.getByText(/nenhum resultado/i)).toBeInTheDocument()
    const atalho = screen.getAllByRole('button', { name: 'Limpar filtros' })
    await userEvent.click(atalho[atalho.length - 1])
    expect(titulos()).toHaveLength(3)
  })

  it('mostra erro e tenta novamente', async () => {
    const carregar = vi.fn().mockRejectedValueOnce(new Error('HTTP 500')).mockResolvedValueOnce(filmes)
    render(<Aba carregar={carregar} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('HTTP 500')
    await userEvent.click(screen.getByRole('button', { name: /tentar novamente/i }))
    expect(await screen.findByText('A Invenção')).toBeInTheDocument()
  })
})
