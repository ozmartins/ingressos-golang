import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import type { Cinema } from './api/tipos'
import App from './App'

const cinemas: Cinema[] = [
  { id: 'c1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A', ativo: true },
  { id: 'c2', nome: 'Cine Norte', cidade: 'Joinville', estado: 'SC', endereco: 'Rua B', ativo: true },
]

const props = {
  carregarFilmes: () => Promise.resolve([]),
  carregarCinemas: () => Promise.resolve(cinemas),
  carregadoresCinema: {
    carregarCinema: (id: string) => Promise.resolve(cinemas.find((c) => c.id === id)!),
    carregarSalas: () => Promise.resolve([]),
    carregarSessoes: () => Promise.resolve([]),
    carregarFilmes: () => Promise.resolve([]),
  },
}

afterEach(() => window.history.replaceState(null, '', '/'))

describe('App', () => {
  it('abre o cinema ao clicar no card e volta preservando aba e filtros', async () => {
    window.history.replaceState(null, '', '/?aba=cinemas')
    render(<App {...props} />)

    await userEvent.type(await screen.findByLabelText('Buscar por nome'), 'Centro')
    await userEvent.click(await screen.findByRole('link', { name: 'Cine Centro' }))

    expect(window.location.pathname).toBe('/cinemas/c1')
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Centro' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('link', { name: /Voltar aos cinemas/ }))

    await waitFor(() => expect(window.location.pathname).toBe('/'))
    expect(window.location.search).toBe('?aba=cinemas')
    expect(screen.queryByRole('heading', { level: 1, name: 'Cine Centro' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Buscar por nome')).toHaveValue('Centro')
    expect(screen.getByRole('tab', { name: 'Cinemas', selected: true })).toBeInTheDocument()
  })

  it('abrir /cinemas/{id} direto mostra a página e "Voltar" leva à aba de cinemas', async () => {
    window.history.replaceState(null, '', '/cinemas/c2')
    render(<App {...props} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Norte' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('link', { name: /Voltar aos cinemas/ }))

    await waitFor(() => expect(window.location.search).toBe('?aba=cinemas'))
    expect(await screen.findByRole('link', { name: 'Cine Centro' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Cinemas', selected: true })).toBeInTheDocument()
  })

  it('na rota inicial mostra a página inicial', async () => {
    render(<App {...props} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Ingressos' })).toBeInTheDocument()
  })
})
