import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ConflitoPoltronas, NaoAutenticado, SessaoIndisponivel } from '../api/estoque'
import type { Cinema, Filme, Poltrona, SessaoNaGrade } from '../api/tipos'
import type { Sessao } from '../dominio/keycloak'
import { PaginaCompra, type CarregadoresCompra } from './PaginaCompra'

const cinema: Cinema = { id: 'c1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A', ativo: true }
const filme: Filme = { id: 'f1', titulo: 'Eu Sou O Rocky', genero: 'Drama', duracao_minutos: 120, classificacao_etaria: '12', status: 'EM_CARTAZ' }
const login: Sessao = { accessToken: 'tok', idToken: null, expiraEm: Date.now() + 60_000 }

const amanha = (dias: number, hora: number) => new Date(Date.now() + dias * 86_400_000 + hora * 3_600_000).toISOString()

const sessaoGrade = (id: string, dias: number, filmeId = 'f1'): SessaoNaGrade => ({
  id,
  filme_id: filmeId,
  filme_titulo: filmeId,
  cinema_id: 'c1',
  cinema_nome: 'Cine Centro',
  sala_numero: 3,
  tipo_tela: '2D',
  data_hora_inicio: amanha(dias, 0),
  idioma: 'DUBLADO',
  preco_base: '30.00',
})

const p = (fileira: string, numero: number, status: Poltrona['status'] = 'LIVRE'): Poltrona => ({
  rotulo: `${fileira}${numero}`,
  fileira,
  numero,
  tipo: 'NORMAL',
  status,
})

const mapaPadrao = [p('A', 1), p('A', 2), p('A', 3, 'OCUPADA')]

function carregadores(extra: Partial<CarregadoresCompra> = {}): CarregadoresCompra {
  return {
    carregarCinema: () => Promise.resolve(cinema),
    carregarFilmes: () => Promise.resolve([filme]),
    carregarSessoes: () => Promise.resolve([sessaoGrade('s2', 3), sessaoGrade('s1', 1), sessaoGrade('x', 1, 'f2')]),
    carregarMapa: () => Promise.resolve(mapaPadrao),
    reservar: () => Promise.resolve({ reserva_id: 'r1', expira_em: new Date(Date.now() + 600_000).toISOString(), mensagem: 'ok' }),
    irParaLogin: () => undefined,
    ...extra,
  }
}

function renderizar(extra: Partial<CarregadoresCompra> = {}, sessao: Sessao | null = login) {
  const c = carregadores(extra)
  render(<PaginaCompra cinemaId="c1" filmeId="f1" sessao={sessao} carregadores={c} />)
  return c
}

const assento = (nome: RegExp) => screen.findByRole('button', { name: nome })
const semEspacoRigido = (t: string | null) => (t ?? '').replace(/\u00a0/g, ' ')

afterEach(() => {
  window.history.replaceState(null, '', '/')
  sessionStorage.clear()
})

describe('PaginaCompra — login', () => {
  it('deslogado: redireciona ao login com retorno e não consulta nada', () => {
    window.history.replaceState(null, '', '/cinemas/c1/filmes/f1/comprar?sessao=s1')
    const irParaLogin = vi.fn()
    const carregarMapa = vi.fn()
    const carregarCinema = vi.fn()
    renderizar({ irParaLogin, carregarMapa, carregarCinema }, null)
    expect(screen.getByRole('status')).toHaveTextContent('Redirecionando para o login')
    expect(irParaLogin).toHaveBeenCalledTimes(1)
    expect(irParaLogin).toHaveBeenCalledWith('/cinemas/c1/filmes/f1/comprar?sessao=s1')
    expect(carregarMapa).not.toHaveBeenCalled()
    expect(carregarCinema).not.toHaveBeenCalled()
  })

  it('logado: não redireciona', async () => {
    const irParaLogin = vi.fn()
    renderizar({ irParaLogin })
    expect(await screen.findByRole('heading', { level: 1, name: 'Eu Sou O Rocky' })).toBeInTheDocument()
    expect(irParaLogin).not.toHaveBeenCalled()
  })

  it('401 ao carregar o mapa refaz o login', async () => {
    const irParaLogin = vi.fn()
    renderizar({ irParaLogin, carregarMapa: () => Promise.reject(new NaoAutenticado()) })
    await waitFor(() => expect(irParaLogin).toHaveBeenCalledTimes(1))
  })
})

describe('PaginaCompra — página', () => {
  it('abre na próxima sessão do filme, com cabeçalho, mapa e resumo vazio', async () => {
    const carregarMapa = vi.fn(() => Promise.resolve(mapaPadrao))
    renderizar({ carregarMapa })
    expect(await assento(/Fileira A, assento 1, livre/)).toBeInTheDocument()
    expect(carregarMapa).toHaveBeenCalledWith('s1', 'tok')
    expect(screen.getByText('Cine Centro - Sala 3')).toBeInTheDocument()
    expect(screen.getByText('Itens').nextSibling).toHaveTextContent('0')
    expect(window.location.search).toBe('?sessao=s1')
  })

  it('usa a sessão da URL e troca de sessão zerando a seleção', async () => {
    window.history.replaceState(null, '', '/x?sessao=s2')
    const carregarMapa = vi.fn(() => Promise.resolve(mapaPadrao))
    renderizar({ carregarMapa })
    await userEvent.click(await assento(/assento 1, livre/))
    expect(screen.getByText('Itens').nextSibling).toHaveTextContent('1')
    await userEvent.selectOptions(screen.getByLabelText('Sessão'), 's1')
    await waitFor(() => expect(carregarMapa).toHaveBeenLastCalledWith('s1', 'tok'))
    expect(await screen.findByText('Nenhum assento selecionado.')).toBeInTheDocument()
    expect(window.location.search).toBe('?sessao=s1')
  })

  it('seleciona, desmarca e calcula o total', async () => {
    renderizar()
    await userEvent.click(await assento(/assento 1, livre/))
    await userEvent.click(screen.getByRole('button', { name: /assento 2, livre/ }))
    expect(screen.getByText('Assentos: A1, A2')).toBeInTheDocument()
    expect(semEspacoRigido(screen.getByText('Total').nextSibling?.textContent ?? '')).toBe('R$ 60,00')
    await userEvent.click(screen.getByRole('button', { name: /assento 1, selecionado/ }))
    expect(semEspacoRigido(screen.getByText('Total').nextSibling?.textContent ?? '')).toBe('R$ 30,00')
  })

  it('avisa ao passar do limite de assentos', async () => {
    const grande = Array.from({ length: 12 }, (_, i) => p('A', i + 1))
    renderizar({ carregarMapa: () => Promise.resolve(grande) })
    for (let i = 1; i <= 10; i++) await userEvent.click(await assento(new RegExp(`assento ${i}, livre`)))
    await userEvent.click(screen.getByRole('button', { name: /assento 11, livre/ }))
    expect(screen.getByRole('alert')).toHaveTextContent('no máximo 10 assentos')
    expect(screen.getByText('Itens').nextSibling).toHaveTextContent('10')
  })

  it('mostra aviso quando o filme não tem sessões futuras', async () => {
    renderizar({ carregarSessoes: () => Promise.resolve([sessaoGrade('x', 1, 'f2')]) })
    expect(await screen.findByText(/Não há sessões disponíveis/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Voltar ao cinema/ })).toBeInTheDocument()
  })

  it('mostra erro de carga com tentar novamente', async () => {
    const carregarCinema = vi.fn().mockRejectedValueOnce(new Error('falhou')).mockResolvedValue(cinema)
    renderizar({ carregarCinema })
    await userEvent.click(await screen.findByRole('button', { name: 'Tentar novamente' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Eu Sou O Rocky' })).toBeInTheDocument()
  })

  it('filme inexistente → não encontrado', async () => {
    renderizar({ carregarFilmes: () => Promise.resolve([]) })
    expect(await screen.findByText(/Filme ou cinema não encontrado/)).toBeInTheDocument()
  })

  it('mapa vazio ou sessão indisponível desabilita a seleção', async () => {
    renderizar({ carregarMapa: () => Promise.reject(new SessaoIndisponivel()) })
    expect(await screen.findByText(/não está disponível para esta sessão/)).toBeInTheDocument()
  })
})

describe('PaginaCompra — reserva', () => {
  it('reserva os assentos e mostra a confirmação com o prazo', async () => {
    const reservar = vi.fn(() => Promise.resolve({ reserva_id: 'r', expira_em: new Date(Date.now() + 600_000).toISOString(), mensagem: 'ok' }))
    renderizar({ reservar })
    await userEvent.click(await assento(/assento 1, livre/))
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }))
    expect(await screen.findByRole('heading', { name: 'Assentos reservados' })).toBeInTheDocument()
    expect(reservar).toHaveBeenCalledWith('s1', ['A1'], 'tok')
    expect(screen.getByText(/Reserva válida até/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Continuar' })).not.toBeInTheDocument()
  })

  it('conflito: avisa, recarrega o mapa e mantém os assentos ainda livres', async () => {
    let chamadas = 0
    const carregarMapa = vi.fn(() => {
      chamadas++
      return Promise.resolve(chamadas === 1 ? mapaPadrao : [p('A', 1, 'OCUPADA'), p('A', 2), p('A', 3, 'OCUPADA')])
    })
    renderizar({ carregarMapa, reservar: () => Promise.reject(new ConflitoPoltronas()) })
    await userEvent.click(await assento(/assento 1, livre/))
    await userEvent.click(screen.getByRole('button', { name: /assento 2, livre/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }))
    expect(await screen.findByText(/acabaram de ser reservados/)).toBeInTheDocument()
    await waitFor(() => expect(carregarMapa).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('Assentos: A2')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /assento 1, ocupado/ })).toBeDisabled()
  })

  it('401 ao reservar guarda a seleção e refaz o login', async () => {
    const irParaLogin = vi.fn()
    renderizar({ irParaLogin, reservar: () => Promise.reject(new NaoAutenticado()) })
    await userEvent.click(await assento(/assento 1, livre/))
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }))
    await waitFor(() => expect(irParaLogin).toHaveBeenCalledTimes(1))
    expect(sessionStorage.getItem('compra:selecao:s1')).toBe('["A1"]')
  })

  it('restaura a seleção guardada ao voltar do login', async () => {
    sessionStorage.setItem('compra:selecao:s1', '["A2"]')
    renderizar()
    expect(await screen.findByText('Assentos: A2')).toBeInTheDocument()
  })

  it('falha de rede mantém a seleção e mostra erro', async () => {
    renderizar({ reservar: () => Promise.reject(new Error('rede')) })
    await userEvent.click(await assento(/assento 1, livre/))
    await userEvent.click(screen.getByRole('button', { name: 'Continuar' }))
    expect(await screen.findByText(/Não foi possível reservar/)).toBeInTheDocument()
    expect(screen.getByText('Assentos: A1')).toBeInTheDocument()
  })

  it('duplo clique envia uma só reserva', async () => {
    let resolver: (v: { reserva_id: string; expira_em: string; mensagem: string }) => void = () => undefined
    const reservar = vi.fn(() => new Promise<{ reserva_id: string; expira_em: string; mensagem: string }>((r) => (resolver = r)))
    renderizar({ reservar })
    await userEvent.click(await assento(/assento 1, livre/))
    const botao = screen.getByRole('button', { name: 'Continuar' })
    await userEvent.dblClick(botao)
    expect(reservar).toHaveBeenCalledTimes(1)
    resolver({ reserva_id: 'r', expira_em: new Date(Date.now() + 1000).toISOString(), mensagem: 'ok' })
    expect(await screen.findByRole('heading', { name: 'Assentos reservados' })).toBeInTheDocument()
  })
})
