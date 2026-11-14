import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Cinema, Filme, SessaoNaGrade } from './api/tipos'
import App from './App'

const cinemas: Cinema[] = [
  { id: 'c1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A', ativo: true },
  { id: 'c2', nome: 'Cine Norte', cidade: 'Joinville', estado: 'SC', endereco: 'Rua B', ativo: true },
]

const props = {
  carregarFilmes: () => Promise.resolve([]),
  carregarCinemas: () => Promise.resolve(cinemas),
  carregadoresCinema: {
    carregarCinema: (id: string) => Promise.resolve(cinemas.find((c) => c.id === id)!),
    carregarSalas: () => Promise.resolve([]),
    carregarSessoes: () => Promise.resolve([]),
    carregarFilmes: () => Promise.resolve([]),
  },
}

const duna: Filme = { id: 'f1', titulo: 'Duna', genero: 'Ficção', duracao_minutos: 120, classificacao_etaria: '12', status: 'EM_CARTAZ' }
const sessaoDuna = (cinemaId: string): SessaoNaGrade => ({
  id: `s-${cinemaId}`,
  filme_id: 'f1',
  filme_titulo: 'Duna',
  cinema_id: cinemaId,
  cinema_nome: cinemaId,
  sala_numero: 1,
  tipo_tela: '2D',
  data_hora_inicio: new Date(Date.now() + 86_400_000).toISOString(),
  idioma: 'DUBLADO',
  preco_base: '30.00',
})
const propsFilme = {
  carregarFilmes: () => Promise.resolve([duna]),
  carregarCinemas: () => Promise.resolve(cinemas),
  carregadoresFilme: {
    carregarFilme: () => Promise.resolve(duna),
    carregarSessoes: () => Promise.resolve([sessaoDuna('c1'), sessaoDuna('c2')]),
    carregarCinemas: () => Promise.resolve(cinemas),
  },
}
const compraSimulada = (irParaLogin = vi.fn()) => ({
  carregarCinema: (id: string) => Promise.resolve(cinemas.find((c) => c.id === id)!),
  carregarFilmes: () => Promise.resolve([duna]),
  carregarSessoes: (cinemaId: string) => Promise.resolve([sessaoDuna(cinemaId)]),
  carregarMapa: () =>
    Promise.resolve([{ rotulo: 'A1', fileira: 'A', numero: 1, tipo: 'NORMAL' as const, status: 'LIVRE' as const }]),
  irParaLogin,
})
const logar = () =>
  sessionStorage.setItem('oidc:sessao', JSON.stringify({ accessToken: 't', idToken: null, expiraEm: Date.now() + 60_000 }))

afterEach(() => {
  window.history.replaceState(null, '', '/')
  sessionStorage.clear()
})

describe('App', () => {
  it('abre o cinema ao clicar no card e volta preservando aba e filtros', async () => {
    window.history.replaceState(null, '', '/?aba=cinemas')
    render(<App {...props} />)

    await userEvent.type(await screen.findByLabelText('Buscar por nome'), 'Centro')
    await userEvent.click(await screen.findByRole('link', { name: 'Cine Centro' }))

    expect(window.location.pathname).toBe('/cinemas/c1')
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Centro' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('link', { name: /Voltar aos cinemas/ }))

    await waitFor(() => expect(window.location.pathname).toBe('/'))
    expect(window.location.search).toBe('?aba=cinemas')
    expect(screen.queryByRole('heading', { level: 1, name: 'Cine Centro' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Buscar por nome')).toHaveValue('Centro')
    expect(screen.getByRole('tab', { name: 'Cinemas', selected: true })).toBeInTheDocument()
  })

  it('abrir /cinemas/{id} direto mostra a página e "Voltar" leva à aba de cinemas', async () => {
    window.history.replaceState(null, '', '/cinemas/c2')
    render(<App {...props} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Cine Norte' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('link', { name: /Voltar aos cinemas/ }))

    await waitFor(() => expect(window.location.search).toBe('?aba=cinemas'))
    expect(await screen.findByRole('link', { name: 'Cine Centro' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Cinemas', selected: true })).toBeInTheDocument()
  })

  it('na rota inicial mostra a página inicial', async () => {
    render(<App {...props} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Ingressos' })).toBeInTheDocument()
  })

  it('compra deslogado: vai ao login com retorno; logado: abre a seleção de assentos', async () => {
    window.history.replaceState(null, '', '/cinemas/c1/filmes/f1/comprar')
    const irParaLogin = vi.fn()
    const compra = {
      carregarCinema: () => Promise.resolve(cinemas[0]),
      carregarFilmes: () =>
        Promise.resolve([{ id: 'f1', titulo: 'Duna', genero: 'Ficção', duracao_minutos: 120, classificacao_etaria: '12', status: 'EM_CARTAZ' as const }]),
      carregarSessoes: () =>
        Promise.resolve([
          {
            id: 's1',
            filme_id: 'f1',
            filme_titulo: 'Duna',
            cinema_id: 'c1',
            cinema_nome: 'Cine Centro',
            sala_numero: 1,
            tipo_tela: '2D' as const,
            data_hora_inicio: new Date(Date.now() + 86_400_000).toISOString(),
            idioma: 'DUBLADO' as const,
            preco_base: '30.00',
          },
        ]),
      carregarMapa: () =>
        Promise.resolve([{ rotulo: 'A1', fileira: 'A', numero: 1, tipo: 'NORMAL' as const, status: 'LIVRE' as const }]),
      irParaLogin,
    }
    const { unmount } = render(<App {...props} carregadoresCompra={compra} />)
    expect(await screen.findByText(/Redirecionando para o login/)).toBeInTheDocument()
    expect(irParaLogin).toHaveBeenCalledWith('/cinemas/c1/filmes/f1/comprar')
    unmount()

    sessionStorage.setItem('oidc:sessao', JSON.stringify({ accessToken: 't', idToken: null, expiraEm: Date.now() + 60_000 }))
    render(<App {...props} carregadoresCompra={compra} />)
    expect(await screen.findByRole('button', { name: /Fileira A, assento 1, livre/ })).toBeInTheDocument()
  })

  it('a rota /pagamento mostra a página de pagamento com a reserva guardada', async () => {
    window.history.replaceState(null, '', '/cinemas/c1/filmes/f1/pagamento')
    sessionStorage.setItem('oidc:sessao', JSON.stringify({ accessToken: 't', idToken: null, expiraEm: Date.now() + 60_000 }))
    sessionStorage.setItem(
      'pagamento:reserva',
      JSON.stringify({
        usuarioId: null,
        cinemaId: 'c1',
        filmeId: 'f1',
        sessaoId: 's1',
        filmeTitulo: 'Rocky',
        posterUrl: null,
        cinemaNome: 'Cine Centro',
        sala: 3,
        inicio: new Date().toISOString(),
        rotulos: ['A1'],
        precoCentavos: 3000,
        taxaCentavos: 0,
        totalCentavos: 3000,
        expiraEm: new Date(Date.now() + 600_000).toISOString(),
      }),
    )
    render(<App {...props} />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Pagamento' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Pagar' })).toBeInTheDocument()
  })

  describe('página do filme', () => {
    it('abre o filme ao clicar no card e volta preservando aba e filtros', async () => {
      render(<App {...props} {...propsFilme} />)
      await userEvent.type(await screen.findByLabelText('Buscar por título'), 'Dun')
      await userEvent.click(await screen.findByRole('link', { name: 'Duna' }))

      expect(window.location.pathname).toBe('/filmes/f1')
      expect(await screen.findByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
      expect(await screen.findByRole('button', { name: 'Comprar Ingressos em Cine Centro' })).toBeInTheDocument()

      await userEvent.click(screen.getByRole('link', { name: /Voltar aos filmes/ }))

      await waitFor(() => expect(window.location.pathname).toBe('/'))
      expect(screen.queryByRole('heading', { level: 1, name: 'Duna' })).not.toBeInTheDocument()
      expect(screen.getByLabelText('Buscar por título')).toHaveValue('Dun')
      expect(screen.getByRole('tab', { name: 'Filmes', selected: true })).toBeInTheDocument()
    })

    it('abrir /filmes/{id} direto mostra a página e "Voltar" leva à aba de filmes', async () => {
      window.history.replaceState(null, '', '/filmes/f1')
      render(<App {...props} {...propsFilme} />)
      expect(await screen.findByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
      await userEvent.click(screen.getByRole('link', { name: /Voltar aos filmes/ }))
      await waitFor(() => expect(window.location.search).toBe('?aba=filmes'))
      expect(await screen.findByRole('link', { name: 'Duna' })).toBeInTheDocument()
    })

    it('logado: "Comprar Ingressos" abre a seleção de assentos e "Voltar" retorna ao filme', async () => {
      logar()
      render(<App {...props} {...propsFilme} carregadoresCompra={compraSimulada()} />)
      await userEvent.click(await screen.findByRole('link', { name: 'Duna' }))
      await userEvent.click(await screen.findByRole('button', { name: 'Comprar Ingressos em Cine Norte' }))

      expect(window.location.pathname).toBe('/cinemas/c2/filmes/f1/comprar')
      expect(await screen.findByRole('button', { name: /Fileira A, assento 1, livre/ })).toBeInTheDocument()

      await userEvent.click(screen.getByRole('link', { name: /Voltar ao cinema/ }))
      await waitFor(() => expect(window.location.pathname).toBe('/filmes/f1'))
      expect(await screen.findByRole('heading', { level: 1, name: 'Duna' })).toBeInTheDocument()
    })

    it('deslogado: "Comprar Ingressos" leva ao login com retorno à compra do filme no cinema', async () => {
      window.history.replaceState(null, '', '/filmes/f1')
      const irParaLogin = vi.fn()
      render(<App {...props} {...propsFilme} carregadoresCompra={compraSimulada(irParaLogin)} />)
      await userEvent.click(await screen.findByRole('button', { name: 'Comprar Ingressos em Cine Centro' }))
      expect(await screen.findByText(/Redirecionando para o login/)).toBeInTheDocument()
      expect(irParaLogin).toHaveBeenCalledWith('/cinemas/c1/filmes/f1/comprar')
    })
  })
})
