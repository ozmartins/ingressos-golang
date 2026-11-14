import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import type { Filme } from '../api/tipos'
import { CartaoFilme } from './CartaoFilme'

const filme: Filme = {
  id: 'f1',
  titulo: 'Duna',
  genero: 'Ficção',
  duracao_minutos: 120,
  classificacao_etaria: '12',
  status: 'EM_CARTAZ',
}

afterEach(() => window.history.replaceState(null, '', '/'))

describe('CartaoFilme', () => {
  it('tem um link com o título apontando para /filmes/{id}', () => {
    render(<ul><CartaoFilme filme={filme} /></ul>)
    expect(screen.getByRole('link', { name: 'Duna' })).toHaveAttribute('href', '/filmes/f1')
  })

  it('clique simples navega sem recarregar', async () => {
    render(<ul><CartaoFilme filme={filme} /></ul>)
    await userEvent.click(screen.getByRole('link', { name: 'Duna' }))
    expect(window.location.pathname).toBe('/filmes/f1')
  })

  it('clique com Ctrl não é interceptado', () => {
    render(<ul><CartaoFilme filme={filme} /></ul>)
    expect(fireEvent.click(screen.getByRole('link', { name: 'Duna' }), { ctrlKey: true })).toBe(true)
    expect(window.location.pathname).toBe('/')
  })
})
