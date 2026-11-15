import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { LinkInicio } from './LinkInicio'

afterEach(() => window.history.replaceState(null, '', '/'))

describe('LinkInicio', () => {
  it('é um link "Página inicial" para "/", com dica e ícone oculto', () => {
    const { container } = render(<LinkInicio />)
    const link = screen.getByRole('link', { name: 'Página inicial' })
    expect(link).toHaveAttribute('href', '/')
    expect(link).toHaveAttribute('title', 'Página inicial')
    expect(container.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
  })

  it('clique simples navega para "/" sem recarregar e empilha no histórico', async () => {
    window.history.replaceState(null, '', '/cinemas/c1')
    const antes = window.history.length
    render(<LinkInicio />)
    await userEvent.click(screen.getByRole('link', { name: 'Página inicial' }))
    expect(window.location.pathname).toBe('/')
    expect(window.history.length).toBe(antes + 1)
  })

  it('em caminho desconhecido leva à raiz', async () => {
    window.history.replaceState(null, '', '/xyz')
    render(<LinkInicio />)
    await userEvent.click(screen.getByRole('link', { name: 'Página inicial' }))
    expect(window.location.pathname).toBe('/')
  })

  it('já na raiz: não cria nova entrada no histórico', async () => {
    window.history.replaceState(null, '', '/')
    const antes = window.history.length
    render(<LinkInicio />)
    await userEvent.click(screen.getByRole('link', { name: 'Página inicial' }))
    expect(window.location.pathname).toBe('/')
    expect(window.history.length).toBe(antes)
  })

  it('Ctrl+clique não é interceptado (padrão do navegador)', () => {
    window.history.replaceState(null, '', '/cinemas/c1')
    render(<LinkInicio />)
    const permitiu = fireEvent.click(screen.getByRole('link', { name: 'Página inicial' }), { ctrlKey: true })
    expect(permitiu).toBe(true)
    expect(window.location.pathname).toBe('/cinemas/c1')
  })

  it('é alcançável por Tab e Enter navega', async () => {
    window.history.replaceState(null, '', '/cinemas/c1')
    render(<LinkInicio />)
    await userEvent.tab()
    expect(screen.getByRole('link', { name: 'Página inicial' })).toHaveFocus()
    await userEvent.keyboard('{Enter}')
    expect(window.location.pathname).toBe('/')
  })
})
