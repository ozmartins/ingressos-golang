import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import type { Cinema } from '../api/tipos'
import { CartaoCinema } from './CartaoCinema'

const base: Cinema = {
  id: '1',
  nome: 'Cine Centro',
  cidade: 'Florianópolis',
  estado: 'SC',
  endereco: 'Rua A, 10',
  ativo: true,
}

describe('CartaoCinema', () => {
  it('exibe o logo a partir da URL remota', () => {
    render(<ul><CartaoCinema cinema={{ ...base, logo_url: 'https://exemplo.com/logo.png' }} /></ul>)
    const logo = screen.getByRole('img', { name: 'Logo de Cine Centro' })
    expect(logo).toHaveAttribute('src', 'https://exemplo.com/logo.png')
  })

  it.each([null, undefined])('usa substituto neutro quando o logo é %s', (logo_url) => {
    render(<ul><CartaoCinema cinema={{ ...base, logo_url }} /></ul>)
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(screen.getByText('Cine Centro')).toBeInTheDocument()
  })

  describe('link para a página do cinema', () => {
    afterEach(() => window.history.replaceState(null, '', '/'))

    it('tem um link com o nome do cinema apontando para /cinemas/{id}', () => {
      render(<ul><CartaoCinema cinema={base} /></ul>)
      expect(screen.getByRole('link', { name: 'Cine Centro' })).toHaveAttribute('href', '/cinemas/1')
    })

    it('clique simples navega sem recarregar', async () => {
      render(<ul><CartaoCinema cinema={base} /></ul>)
      await userEvent.click(screen.getByRole('link', { name: 'Cine Centro' }))
      expect(window.location.pathname).toBe('/cinemas/1')
    })

    it('clique com Ctrl não é interceptado', () => {
      render(<ul><CartaoCinema cinema={base} /></ul>)
      const link = screen.getByRole('link', { name: 'Cine Centro' })
      // fireEvent.click devolve false quando alguém chamou preventDefault.
      expect(fireEvent.click(link, { ctrlKey: true })).toBe(true)
      expect(window.location.pathname).toBe('/')
    })
  })
})
