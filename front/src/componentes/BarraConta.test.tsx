import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { BarraConta } from './BarraConta'

const sessao = { accessToken: 't', idToken: null, expiraEm: Date.now() + 60_000 }

describe('BarraConta', () => {
  it('deslogado: mostra Criar Conta e Login, sem Log out', () => {
    render(<BarraConta sessao={null} onSair={() => {}} />)
    expect(screen.getByRole('button', { name: 'Criar Conta' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Login' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Log out' })).toBeNull()
  })

  it('logado: troca Login por Log out', async () => {
    const onSair = vi.fn()
    render(<BarraConta sessao={sessao} onSair={onSair} />)
    expect(screen.queryByRole('button', { name: 'Login' })).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Log out' }))
    expect(onSair).toHaveBeenCalled()
  })
})
