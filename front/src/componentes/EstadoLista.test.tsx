import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { EstadoLista } from './EstadoLista'

const base = {
  erro: '',
  total: 3,
  visiveis: 3,
  rotulo: 'filmes',
  onTentarNovamente: () => {},
  onLimparFiltros: () => {},
}

describe('EstadoLista', () => {
  it('mostra carregamento', () => {
    render(<EstadoLista {...base} estado="carregando">conteúdo</EstadoLista>)
    expect(screen.getByRole('status')).toHaveTextContent(/carregando filmes/i)
    expect(screen.queryByText('conteúdo')).not.toBeInTheDocument()
  })

  it('mostra erro com nova tentativa', async () => {
    const tentar = vi.fn()
    render(<EstadoLista {...base} estado="erro" erro="HTTP 500" onTentarNovamente={tentar}>x</EstadoLista>)
    expect(screen.getByRole('alert')).toHaveTextContent('HTTP 500')
    await userEvent.click(screen.getByRole('button', { name: /tentar novamente/i }))
    expect(tentar).toHaveBeenCalledOnce()
  })

  it('mostra lista vazia quando não há itens cadastrados', () => {
    render(<EstadoLista {...base} estado="pronto" total={0} visiveis={0}>x</EstadoLista>)
    expect(screen.getByText(/ainda não há filmes/i)).toBeInTheDocument()
  })

  it('mostra sem resultado com atalho para limpar', async () => {
    const limpar = vi.fn()
    render(<EstadoLista {...base} estado="pronto" visiveis={0} onLimparFiltros={limpar}>x</EstadoLista>)
    expect(screen.getByText(/nenhum resultado/i)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /limpar filtros/i }))
    expect(limpar).toHaveBeenCalledOnce()
  })

  it('renderiza os filhos quando há itens', () => {
    render(<EstadoLista {...base} estado="pronto">conteúdo</EstadoLista>)
    expect(screen.getByText('conteúdo')).toBeInTheDocument()
  })
})
