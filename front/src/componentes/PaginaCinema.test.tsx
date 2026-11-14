import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { CinemaNaoEncontrado } from '../api/catalogo'
import type { Cinema, Filme, Sala, SessaoNaGrade } from '../api/tipos'
import { PaginaCinema, type CarregadoresCinema } from './PaginaCinema'

const cinema: Cinema = {
  id: 'c1',
  nome: 'Cine Centro',
  cidade: 'Florianópolis',
  estado: 'SC',
  endereco: 'Rua A, 10',
  logo_url: 'https://exemplo.com/logo.png',
  ativo: true,
}

const filme = (id: string, titulo: string, status: Filme['status'] = 'EM_CARTAZ'): Filme => ({
  id,
  titulo,
  genero: 'Ficção',
  duracao_minutos: 120,
  classificacao_etaria: '12',
  status,
})

const sala = (numero: number): Sala => ({
  id: `s${numero}`,
  cinema_id: 'c1',
  numero,
  tipo_tela: '2D',
  capacidade_total: 100,
  ativo: true,
})

function sessao(id: string, filmeId: string, dia: number, hora: number, salaNumero = 1): SessaoNaGrade {
  return {
    id,
    filme_id: filmeId,
    filme_titulo: filmeId,
    cinema_id: 'c1',
    cinema_nome: 'Cine Centro',
    sala_numero: salaNumero,
    tipo_tela: '2D',
    data_hora_inicio: new Date(2026, 10, dia, hora, 0).toISOString(),
    idioma: 'DUBLADO',
    preco_base: '30.00',
  }
}

function carregadores(extra: Partial<CarregadoresCinema> = {}): CarregadoresCinema {
  return {
    carregarCinema: () => Promise.resolve(cinema),
    carregarSalas: () => Promise.resolve([sala(1), sala(2)]),
    carregarSessoes: () =>
      Promise.resolve([
        sessao('a', 'f1', 14, 18),
        sessao('b', 'f1', 15, 20),
        sessao('c', 'f2', 14, 19),
        sessao('d', 'f3', 14, 21, 2),
        sessao('e', 'f4', 14, 22, 2),
      ]),
    carregarFilmes: () =>
      Promise.resolve([
        filme('f1', 'Duna'),
        filme('f2', 'Avatar'),
        filme('f3', 'Em Breve Filme', 'BREVE'),
        filme('f4', 'Antigo', 'FORA_DE_CARTAZ'),
      ]),
    ...extra,
  }
}

afterEach(() => window.history.replaceState(null, '', '/'))

describe('PaginaCinema', () => {
  it('mostra os dados do cinema, as salas e a programação agrupada por filme', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores()} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Centro' })).toBeInTheDocument()
    expect(screen.getByText('Florianópolis / SC')).toBeInTheDocument()
    expect(screen.getByText('Rua A, 10')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Logo de Cine Centro' })).toBeInTheDocument()
    expect(await screen.findByRole('heading', { name: 'Duna' })).toBeInTheDocument()
    expect(screen.getAllByRole('heading', { name: 'Duna' })).toHaveLength(1)
    expect(screen.getByText('Sala 1')).toBeInTheDocument()
    expect(screen.getByText('Sala 2')).toBeInTheDocument()
  })

  it('FR-013: só filmes em cartaz; sala só com filmes fora de cartaz fica sem sessões', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores()} />)
    await screen.findByRole('heading', { name: 'Duna' })
    expect(screen.queryByText('Em Breve Filme')).not.toBeInTheDocument()
    expect(screen.queryByText('Antigo')).not.toBeInTheDocument()
    expect(screen.getByText('Sem sessões programadas')).toBeInTheDocument()
  })

  it('filtra por dia e restaura em "Todos os dias"', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores()} />)
    await screen.findByRole('heading', { name: 'Duna' })
    const seletor = screen.getByLabelText('Dia')
    await userEvent.selectOptions(seletor, '2026-11-15')
    expect(screen.getByRole('heading', { name: 'Duna' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Avatar' })).not.toBeInTheDocument()
    await userEvent.selectOptions(seletor, '')
    expect(screen.getByRole('heading', { name: 'Avatar' })).toBeInTheDocument()
  })

  it('"Comprar Ingressos" abre a página de compra do filme', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores()} />)
    await screen.findByRole('heading', { name: 'Duna' })
    await userEvent.click(screen.getAllByRole('button', { name: 'Comprar Ingressos' })[0])
    expect(window.location.pathname).toBe('/cinemas/c1/filmes/f1/comprar')
  })

  it('cinema sem sessões em cartaz mostra mensagem', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarSessoes: () => Promise.resolve([]) })} />)
    expect(await screen.findByText(/não tem sessões de filmes em cartaz/)).toBeInTheDocument()
    expect(screen.getAllByText('Sem sessões programadas')).toHaveLength(2)
  })

  it('cinema sem salas mostra mensagem', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarSalas: () => Promise.resolve([]) })} />)
    expect(await screen.findByText('Este cinema ainda não tem salas cadastradas.')).toBeInTheDocument()
  })

  it('mostra carregando', () => {
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarCinema: () => new Promise(() => {}) })} />)
    expect(screen.getByRole('status')).toHaveTextContent('Carregando cinema')
  })

  it.each([
    ['inexistente', () => Promise.reject(new CinemaNaoEncontrado())],
    ['inativo', () => Promise.resolve({ ...cinema, ativo: false })],
  ])('cinema %s mostra "não encontrado" com caminho de volta', async (_, carregarCinema) => {
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarCinema })} />)
    expect(await screen.findByRole('heading', { name: 'Cinema não encontrado' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Voltar aos cinemas/ })).toHaveAttribute('href', '/?aba=cinemas')
  })

  it('erro de consulta oferece "Tentar novamente"', async () => {
    const carregarCinema = vi.fn().mockRejectedValueOnce(new Error('falhou')).mockResolvedValueOnce(cinema)
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarCinema })} />)
    await userEvent.click(await screen.findByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Centro' })).toBeInTheDocument()
  })

  it('falha na listagem de filmes vira erro da programação, sem derrubar o cinema', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores({ carregarFilmes: () => Promise.reject(new Error('x')) })} />)
    expect(await screen.findByText('Não foi possível carregar a programação.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Cine Centro' })).toBeInTheDocument()
  })

  it('"Voltar" sem histórico interno navega para a aba de cinemas', async () => {
    render(<PaginaCinema id="c1" carregadores={carregadores()} />)
    await screen.findByRole('heading', { level: 1 })
    await userEvent.click(screen.getByRole('link', { name: /Voltar aos cinemas/ }))
    await waitFor(() => expect(window.location.search).toBe('?aba=cinemas'))
  })
})
