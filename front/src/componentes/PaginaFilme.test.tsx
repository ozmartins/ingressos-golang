import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FilmeNaoEncontrado } from '../api/catalogo'
import type { Cinema, Filme, SessaoNaGrade } from '../api/tipos'
import { PaginaFilme, type CarregadoresFilme } from './PaginaFilme'

const filme: Filme = {
  id: 'f1',
  titulo: 'Duna',
  genero: 'Ficção',
  duracao_minutos: 125,
  classificacao_etaria: '12',
  status: 'EM_CARTAZ',
}

const cinema = (id: string, nome: string, cidade: string, estado: string): Cinema => ({
  id,
  nome,
  cidade,
  estado,
  endereco: `Rua ${nome}`,
  ativo: true,
})

const cinemas = [
  cinema('c1', 'Cine Centro', 'Florianópolis', 'SC'),
  cinema('c2', 'Cine Norte', 'Joinville', 'SC'),
  cinema('c3', 'Cine Sul', 'Curitiba', 'PR'),
  cinema('c4', 'Cine Sem Duna', 'Porto Alegre', 'RS'),
]

const sessao = (id: string, cinemaId: string, deslocamentoMs = 86_400_000): SessaoNaGrade => ({
  id,
  filme_id: 'f1',
  filme_titulo: 'Duna',
  cinema_id: cinemaId,
  cinema_nome: cinemaId,
  sala_numero: 1,
  tipo_tela: '2D',
  data_hora_inicio: new Date(Date.now() + deslocamentoMs).toISOString(),
  idioma: 'DUBLADO',
  preco_base: '30.00',
})

function carregadores(extra: Partial<CarregadoresFilme> = {}): CarregadoresFilme {
  return {
    carregarFilme: () => Promise.resolve(filme),
    carregarSessoes: () =>
      Promise.resolve([sessao('s1', 'c1'), sessao('s2', 'c2'), sessao('s3', 'c3'), sessao('s4', 'c4', -3_600_000)]),
    carregarCinemas: () => Promise.resolve(cinemas),
    ...extra,
  }
}

afterEach(() => window.history.replaceState(null, '', '/'))

describe('PaginaFilme', () => {
  it('mostra os dados do filme e só os cinemas com sessão futura dele', async () => {
    render(<PaginaFilme id="f1" carregadores={carregadores()} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
    expect(screen.getByText('Ficção · 2h 05min')).toBeInTheDocument()
    expect(screen.getByText('Classificação: 12')).toBeInTheDocument()
    expect(await screen.findByText('Cine Centro')).toBeInTheDocument()
    expect(screen.getByText('Florianópolis / SC')).toBeInTheDocument()
    expect(screen.getByText('Rua Cine Centro')).toBeInTheDocument()
    expect(screen.getByText('Cine Norte')).toBeInTheDocument()
    expect(screen.getByText('Cine Sul')).toBeInTheDocument()
    expect(screen.queryByText('Cine Sem Duna')).not.toBeInTheDocument()
    expect(screen.getByText('3 cinemas')).toBeInTheDocument()
  })

  it('mostra carregando e depois a lista', async () => {
    render(<PaginaFilme id="f1" carregadores={carregadores()} />)
    expect(screen.getByRole('status')).toHaveTextContent('Carregando filme')
    expect(await screen.findByText('Cine Centro')).toBeInTheDocument()
  })

  it('filme inexistente informa e oferece voltar', async () => {
    const carregarFilme = () => Promise.reject(new FilmeNaoEncontrado())
    render(<PaginaFilme id="x" carregadores={carregadores({ carregarFilme })} />)
    expect(await screen.findByRole('heading', { name: 'Filme não encontrado' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Voltar aos filmes/ })).toBeInTheDocument()
  })

  it('falha ao carregar o filme oferece tentar novamente', async () => {
    const carregarFilme = vi.fn().mockRejectedValueOnce(new Error('rede')).mockResolvedValue(filme)
    render(<PaginaFilme id="f1" carregadores={carregadores({ carregarFilme })} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('Não foi possível carregar o filme')
    await userEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
  })

  it('falha ao carregar os cinemas mantém o filme e oferece tentar novamente', async () => {
    const carregarCinemas = vi.fn().mockRejectedValueOnce(new Error('rede')).mockResolvedValue(cinemas)
    render(<PaginaFilme id="f1" carregadores={carregadores({ carregarCinemas })} />)
    expect(await screen.findByText('Não foi possível carregar os cinemas.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByText('Cine Centro')).toBeInTheDocument()
  })

  it('filme sem cinemas em cartaz mostra a mensagem, sem filtros', async () => {
    render(<PaginaFilme id="f1" carregadores={carregadores({ carregarSessoes: () => Promise.resolve([]) })} />)
    expect(await screen.findByText('Este filme não está em cartaz em nenhum cinema.')).toBeInTheDocument()
    expect(screen.queryByRole('search')).not.toBeInTheDocument()
  })

  it('filme em breve mostra a situação e a mensagem de sem cinemas', async () => {
    const breve: Filme = { ...filme, status: 'BREVE' }
    render(
      <PaginaFilme
        id="f1"
        carregadores={carregadores({ carregarFilme: () => Promise.resolve(breve), carregarSessoes: () => Promise.resolve([]) })}
      />,
    )
    expect(await screen.findByText('Em breve')).toBeInTheDocument()
    expect(screen.getByText('Este filme não está em cartaz em nenhum cinema.')).toBeInTheDocument()
  })

  it('"Voltar aos filmes" sem histórico interno leva à aba de filmes', async () => {
    window.history.replaceState(null, '', '/filmes/f1')
    render(<PaginaFilme id="f1" carregadores={carregadores()} />)
    await userEvent.click(await screen.findByRole('link', { name: /Voltar aos filmes/ }))
    expect(window.location.pathname + window.location.search).toBe('/?aba=filmes')
  })

  describe('filtros por UF e cidade', () => {
    it('UF restringe cinemas e cidades; cidade restringe mais', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')

      await userEvent.selectOptions(screen.getByLabelText('UF'), 'SC')
      expect(screen.queryByText('Cine Sul')).not.toBeInTheDocument()
      expect(screen.getByText('2 cinemas')).toBeInTheDocument()
      const cidades = within(screen.getByLabelText('Cidade')).getAllByRole('option').map((o) => o.textContent)
      expect(cidades).toEqual(['Todas as cidades', 'Florianópolis', 'Joinville'])

      await userEvent.selectOptions(screen.getByLabelText('Cidade'), 'Joinville')
      expect(screen.getByText('1 cinema')).toBeInTheDocument()
      expect(screen.getByText('Cine Norte')).toBeInTheDocument()
      expect(screen.queryByText('Cine Centro')).not.toBeInTheDocument()
    })

    it('só oferece UFs que têm o filme em cartaz', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      const ufs = within(screen.getByLabelText('UF')).getAllByRole('option').map((o) => o.textContent)
      expect(ufs).toEqual(['Todas as UFs', 'PR', 'SC'])
    })

    it('trocar a UF descarta a cidade incompatível', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      await userEvent.selectOptions(screen.getByLabelText('Cidade'), 'Joinville')
      await userEvent.selectOptions(screen.getByLabelText('UF'), 'PR')
      expect(screen.getByLabelText('Cidade')).toHaveValue('')
      expect(screen.getByText('Cine Sul')).toBeInTheDocument()
      expect(screen.getByText('1 cinema')).toBeInTheDocument()
    })

    it('"Limpar filtros" restaura a lista e fica desabilitado sem filtros', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      const limpar = screen.getByRole('button', { name: 'Limpar filtros' })
      expect(limpar).toBeDisabled()
      await userEvent.selectOptions(screen.getByLabelText('UF'), 'PR')
      expect(limpar).toBeEnabled()
      await userEvent.click(limpar)
      expect(screen.getByText('3 cinemas')).toBeInTheDocument()
      expect(limpar).toBeDisabled()
    })

    it('cidade sem UF escolhida filtra; combinação sem resultado mostra mensagem com limpar', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      await userEvent.selectOptions(screen.getByLabelText('Cidade'), 'Curitiba')
      expect(screen.getByText('1 cinema')).toBeInTheDocument()
      await userEvent.selectOptions(screen.getByLabelText('UF'), 'SC')
      // Curitiba não é de SC: a cidade é descartada.
      expect(screen.getByLabelText('Cidade')).toHaveValue('')
      expect(screen.getByText('2 cinemas')).toBeInTheDocument()
    })
  })

  describe('Comprar Ingressos', () => {
    it('cada cinema tem o botão e ele abre a compra do filme naquele cinema', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      expect(screen.getAllByRole('button', { name: /Comprar Ingressos/ })).toHaveLength(3)
      await userEvent.click(screen.getByRole('button', { name: 'Comprar Ingressos em Cine Norte' }))
      expect(window.location.pathname).toBe('/cinemas/c2/filmes/f1/comprar')
    })

    it('o botão respeita o filtro aplicado', async () => {
      render(<PaginaFilme id="f1" carregadores={carregadores()} />)
      await screen.findByText('Cine Centro')
      await userEvent.selectOptions(screen.getByLabelText('UF'), 'PR')
      await waitFor(() => expect(screen.getAllByRole('button', { name: /Comprar Ingressos/ })).toHaveLength(1))
      await userEvent.click(screen.getByRole('button', { name: /Comprar Ingressos/ }))
      expect(window.location.pathname).toBe('/cinemas/c3/filmes/f1/comprar')
    })
  })
})
