import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
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
})
