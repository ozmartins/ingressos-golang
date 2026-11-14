import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { FormularioCartao } from './FormularioCartao'

const preencher = async (numero = '4111111111111111') => {
  await userEvent.type(screen.getByLabelText('Número do cartão'), numero)
  await userEvent.type(screen.getByLabelText('Nome impresso no cartão'), 'Maria Souza')
  await userEvent.type(screen.getByLabelText('Validade (MM/AA)'), '1230')
  await userEvent.type(screen.getByLabelText('Código de segurança'), '123')
}

describe('FormularioCartao', () => {
  it('aplica máscaras enquanto digita', async () => {
    render(<FormularioCartao onPagar={() => undefined} desabilitado={false} processando={false} />)
    await preencher()
    expect(screen.getByLabelText('Número do cartão')).toHaveValue('4111 1111 1111 1111')
    expect(screen.getByLabelText('Validade (MM/AA)')).toHaveValue('12/30')
  })

  it('em branco: mostra erro por campo, foca o primeiro e não paga', async () => {
    const onPagar = vi.fn()
    render(<FormularioCartao onPagar={onPagar} desabilitado={false} processando={false} />)
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    expect(onPagar).not.toHaveBeenCalled()
    expect(screen.getByText('Informe o número do cartão.')).toBeInTheDocument()
    expect(screen.getByText('Informe o nome impresso no cartão.')).toBeInTheDocument()
    const numero = screen.getByLabelText('Número do cartão')
    expect(numero).toHaveAttribute('aria-invalid', 'true')
    expect(numero).toHaveAccessibleDescription('Informe o número do cartão.')
    expect(numero).toHaveFocus()
  })

  it('com dados válidos chama onPagar', async () => {
    const onPagar = vi.fn()
    render(<FormularioCartao onPagar={onPagar} desabilitado={false} processando={false} />)
    await preencher()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    expect(onPagar).toHaveBeenCalledWith({ numero: '4111 1111 1111 1111', nome: 'Maria Souza', validade: '12/30', cvv: '123' })
  })

  it('limpa o erro do campo ao editá-lo e mostra o aviso de simulação', async () => {
    render(<FormularioCartao onPagar={() => undefined} desabilitado={false} processando={false} />)
    expect(screen.getByText(/Não informe dados reais/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await userEvent.type(screen.getByLabelText('Número do cartão'), '4')
    expect(screen.queryByText('Informe o número do cartão.')).not.toBeInTheDocument()
  })

  it('desabilita tudo durante o processamento', () => {
    render(<FormularioCartao onPagar={() => undefined} desabilitado={false} processando />)
    expect(screen.getByRole('button', { name: 'Processando…' })).toBeDisabled()
    expect(screen.getByLabelText('Número do cartão')).toBeDisabled()
  })
})
