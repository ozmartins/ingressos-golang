import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { SessaoExibida } from '../dominio/programacao'
import { ProgramacaoFilme } from './ProgramacaoFilme'

const filme = {
  id: 'f',
  titulo: 'Duna',
  genero: 'Ficção',
  duracao_minutos: 155,
  classificacao_etaria: '14',
  status: 'EM_CARTAZ' as const,
}

const sessao = (id: string, hora: number, sala: number): SessaoExibida => ({
  id,
  filme,
  inicio: new Date(2026, 10, 14, hora, 0),
  dia: '2026-11-14',
  salaNumero: sala,
  tipoTela: '3D',
  idioma: 'LEGENDADO',
  preco: 42,
})

describe('ProgramacaoFilme', () => {
  it('mostra o filme uma vez com os horários, salas, idioma e preço', () => {
    render(<ul><ProgramacaoFilme programacao={{ filme, sessoes: [sessao('a', 18, 1), sessao('b', 21, 2)] }} /></ul>)
    expect(screen.getAllByRole('heading', { name: 'Duna' })).toHaveLength(1)
    expect(screen.getByText(/Ficção · 2h 35min · Classificação: 14/)).toBeInTheDocument()
    const itens = screen.getAllByRole('listitem').filter((li) => li.classList.contains('sessao'))
    expect(itens).toHaveLength(2)
    expect(within(itens[0]).getByText(/18:00/)).toBeInTheDocument()
    expect(within(itens[0]).getByText('Sala 1 · 3D')).toBeInTheDocument()
    expect(within(itens[0]).getByText('Legendado')).toBeInTheDocument()
    expect(within(itens[1]).getByText(/R\$\s42,00/)).toBeInTheDocument()
  })

  it('exibe o botão "Comprar Ingressos" e o aciona com o filme', async () => {
    const aoComprar = vi.fn()
    render(<ul><ProgramacaoFilme programacao={{ filme, sessoes: [sessao('a', 18, 1)] }} onComprar={aoComprar} /></ul>)
    await userEvent.click(screen.getByRole('button', { name: 'Comprar Ingressos' }))
    expect(aoComprar).toHaveBeenCalledWith(filme)
  })

  it('um botão por filme', () => {
    render(<ul><ProgramacaoFilme programacao={{ filme, sessoes: [sessao('a', 18, 1), sessao('b', 20, 1)] }} /></ul>)
    expect(screen.getAllByRole('button', { name: 'Comprar Ingressos' })).toHaveLength(1)
  })
})
