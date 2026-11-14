import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { Filme, SessaoNaGrade } from '../api/tipos'
import { calcularPedido } from '../dominio/pedido'
import { ResumoPedido } from './ResumoPedido'

const filme: Filme = { id: 'f1', titulo: 'Eu Sou O Rocky', genero: 'Drama', duracao_minutos: 120, classificacao_etaria: '12', status: 'EM_CARTAZ' }
const sessao: SessaoNaGrade = {
  id: 's1',
  filme_id: 'f1',
  filme_titulo: 'Eu Sou O Rocky',
  cinema_id: 'c1',
  cinema_nome: 'Cine Centro',
  sala_numero: 3,
  tipo_tela: '2D',
  data_hora_inicio: new Date(2026, 10, 14, 19, 0).toISOString(),
  idioma: 'LEGENDADO',
  preco_base: '30.00',
}

const semEspacoRigido = (t: string | null) => (t ?? '').replace(/\u00a0/g, ' ')

function renderizar(rotulos: string[], extra: { reservando?: boolean; reservado?: boolean; aoContinuar?: () => void } = {}) {
  render(
    <ResumoPedido
      filme={filme}
      sessao={sessao}
      selecionados={rotulos}
      pedido={calcularPedido(3000, rotulos)}
      reservando={extra.reservando ?? false}
      reservado={extra.reservado ?? false}
      onContinuar={extra.aoContinuar ?? (() => undefined)}
    />,
  )
}

describe('ResumoPedido', () => {
  it('sem assentos: 0 itens, total zerado e Continuar desabilitado', () => {
    renderizar([])
    expect(screen.getByRole('heading', { name: 'Resumo do pedido' })).toBeInTheDocument()
    expect(screen.getByText('Eu Sou O Rocky')).toBeInTheDocument()
    expect(screen.getByText('Sala 3')).toBeInTheDocument()
    expect(screen.getByText('Itens').nextSibling).toHaveTextContent('0')
    expect(semEspacoRigido(screen.getByText('Total').nextSibling?.textContent ?? '')).toBe('R$ 0,00')
    expect(screen.getByRole('button', { name: 'Continuar' })).toBeDisabled()
  })

  it('com assentos: lista, itens e total', async () => {
    const aoContinuar = vi.fn()
    renderizar(['A1', 'A2'], { aoContinuar })
    expect(screen.getByText('Assentos: A1, A2')).toBeInTheDocument()
    expect(screen.getByText('Itens').nextSibling).toHaveTextContent('2')
    expect(semEspacoRigido(screen.getByText('Total').nextSibling?.textContent ?? '')).toBe('R$ 60,00')
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }))
    expect(aoContinuar).toHaveBeenCalled()
  })

  it('reservando desabilita e reservado remove a ação', () => {
    renderizar(['A1'], { reservando: true })
    expect(screen.getByRole('button', { name: 'Reservando…' })).toBeDisabled()
  })

  it('reservado esconde Continuar', () => {
    renderizar(['A1'], { reservado: true })
    expect(screen.queryByRole('button', { name: /Continuar/ })).not.toBeInTheDocument()
  })
})
