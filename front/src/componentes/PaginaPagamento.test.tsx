import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Sessao } from '../dominio/keycloak'
import { lerReserva, salvarReserva, type ReservaPagamento } from '../dominio/reservaPagamento'
import { PaginaPagamento } from './PaginaPagamento'

const login: Sessao = { accessToken: 'tok', idToken: null, expiraEm: Date.now() + 60_000 }
const AGORA = new Date('2026-11-15T21:00:00.000Z').getTime()

const reserva: ReservaPagamento = {
  usuarioId: null,
  cinemaId: 'c1',
  filmeId: 'f1',
  sessaoId: 's1',
  filmeTitulo: 'Rocky',
  posterUrl: null,
  cinemaNome: 'Cine Centro',
  sala: 3,
  inicio: '2026-11-15T22:00:00.000Z',
  rotulos: ['A1', 'A2'],
  precoCentavos: 3000,
  taxaCentavos: 0,
  totalCentavos: 6000,
  expiraEm: new Date(AGORA + 600_000).toISOString(),
}

let relogio = AGORA

function renderizar(sessao: Sessao | null = login, irParaLogin: (r: string) => void = () => undefined) {
  render(<PaginaPagamento cinemaId="c1" filmeId="f1" sessao={sessao} irParaLogin={irParaLogin} atrasoMs={2000} agora={() => relogio} />)
}

const preencher = async (numero = '4111111111111111') => {
  await userEvent.type(screen.getByLabelText('Número do cartão'), numero)
  await userEvent.type(screen.getByLabelText('Nome impresso no cartão'), 'Maria Souza')
  await userEvent.type(screen.getByLabelText('Validade (MM/AA)'), '1230')
  await userEvent.type(screen.getByLabelText('Código de segurança'), '123')
}

const avancar = (ms: number) =>
  act(async () => {
    relogio += ms
    await vi.advanceTimersByTimeAsync(ms)
  })

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true, now: AGORA })
  relogio = AGORA
})

afterEach(() => {
  vi.useRealTimers()
  window.history.replaceState(null, '', '/')
  sessionStorage.clear()
  localStorage.clear()
})

describe('PaginaPagamento — abertura', () => {
  it('mostra o resumo, a contagem e o formulário', () => {
    salvarReserva(reserva)
    renderizar()
    expect(screen.getByRole('heading', { level: 1, name: 'Pagamento' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Rocky' })).toBeInTheDocument()
    expect(screen.getByText('Assentos: A1, A2')).toBeInTheDocument()
    expect(screen.getByRole('timer')).toHaveTextContent('10:00')
    expect(screen.getByRole('button', { name: 'Pagar' })).toBeEnabled()
  })

  it('"Voltar" leva à seleção de assentos da sessão', async () => {
    salvarReserva(reserva)
    renderizar()
    await userEvent.click(screen.getByRole('link', { name: /Voltar aos assentos/ }))
    expect(window.location.pathname + window.location.search).toBe('/cinemas/c1/filmes/f1/comprar?sessao=s1')
  })

  it('sem reserva: avisa e oferece caminhos', () => {
    renderizar()
    expect(screen.getByRole('heading', { name: 'Nenhuma reserva em andamento' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Escolher assentos' })).toHaveAttribute('href', '/cinemas/c1/filmes/f1/comprar')
    expect(screen.getByRole('link', { name: 'Página inicial' })).toHaveAttribute('href', '/')
  })

  it('deslogado: redireciona ao login com retorno ao endereço atual', () => {
    window.history.replaceState(null, '', '/cinemas/c1/filmes/f1/pagamento')
    const irParaLogin = vi.fn()
    renderizar(null, irParaLogin)
    expect(irParaLogin).toHaveBeenCalledWith('/cinemas/c1/filmes/f1/pagamento')
    expect(screen.getByText('Redirecionando para o login…')).toBeInTheDocument()
  })
})

describe('PaginaPagamento — dono da reserva', () => {
  const token = (sub: string) => `x.${btoa(JSON.stringify({ sub }))}.y`

  it('mostra a reserva ao mesmo usuário e a esconde de outro', () => {
    salvarReserva({ ...reserva, usuarioId: 'u1' })
    renderizar({ ...login, accessToken: token('u1') })
    expect(screen.getByRole('heading', { name: 'Rocky' })).toBeInTheDocument()
    document.body.innerHTML = ''
    renderizar({ ...login, accessToken: token('u2') })
    expect(screen.getByRole('heading', { name: 'Nenhuma reserva em andamento' })).toBeInTheDocument()
  })
})

describe('PaginaPagamento — pagar', () => {
  it('em branco: erros e nada é processado', async () => {
    salvarReserva(reserva)
    renderizar()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    expect(screen.getByText('Informe o número do cartão.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Processando…' })).not.toBeInTheDocument()
  })

  it('validade vencida mostra o erro do campo', async () => {
    salvarReserva(reserva)
    renderizar()
    await userEvent.type(screen.getByLabelText('Número do cartão'), '123')
    await userEvent.type(screen.getByLabelText('Nome impresso no cartão'), 'Maria Souza')
    await userEvent.type(screen.getByLabelText('Validade (MM/AA)'), '0120')
    await userEvent.type(screen.getByLabelText('Código de segurança'), '123')
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    expect(screen.getByText('Cartão vencido.')).toBeInTheDocument()
  })

  it('aceita qualquer número de cartão (ex.: curto e sem dígito verificador)', async () => {
    salvarReserva(reserva)
    renderizar()
    await preencher('12345')
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    expect(screen.getByRole('heading', { name: 'Pagamento aprovado' })).toBeInTheDocument()
  })

  it('dados válidos: processando → aprovado e a reserva é removida', async () => {
    salvarReserva(reserva)
    renderizar()
    await preencher()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    expect(screen.getByRole('button', { name: 'Processando…' })).toBeDisabled()
    await avancar(2000)
    expect(screen.getByRole('heading', { name: 'Pagamento aprovado' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('A1, A2')
    expect(screen.queryByRole('timer')).not.toBeInTheDocument()
    expect(lerReserva('c1', 'f1', null)).toBeNull()
    expect(screen.getByRole('link', { name: 'Voltar à página inicial' })).toHaveAttribute('href', '/')
  })

  it('cartão terminado em 0000 é recusado e permite tentar outro', async () => {
    salvarReserva(reserva)
    renderizar()
    await preencher('4111111000030000')
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    expect(screen.getByRole('heading', { name: 'Pagamento recusado' })).toBeInTheDocument()
    expect(lerReserva('c1', 'f1', null)).not.toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Tentar outro cartão' }))
    expect(screen.getByRole('button', { name: 'Pagar' })).toBeEnabled()
    expect(screen.getByRole('heading', { name: 'Rocky' })).toBeInTheDocument()
  })

  it('duplo clique processa uma só vez', async () => {
    salvarReserva(reserva)
    renderizar()
    await preencher()
    await userEvent.dblClick(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    expect(screen.getAllByRole('heading', { name: 'Pagamento aprovado' })).toHaveLength(1)
  })

  it('não faz rede, não guarda dados de cartão e não escreve no console', async () => {
    const fetch = vi.fn()
    vi.stubGlobal('fetch', fetch)
    const log = vi.spyOn(console, 'log')
    const info = vi.spyOn(console, 'info')
    const warn = vi.spyOn(console, 'warn')
    const erro = vi.spyOn(console, 'error')
    salvarReserva(reserva)
    renderizar()
    await preencher()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    expect(fetch).not.toHaveBeenCalled()
    const guardado = JSON.stringify({ ...sessionStorage }) + JSON.stringify({ ...localStorage })
    expect(guardado).not.toContain('4111')
    expect(guardado).not.toContain('Maria')
    for (const espiao of [log, info, warn, erro]) expect(espiao).not.toHaveBeenCalled()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })
})

describe('PaginaPagamento — prazo', () => {
  it('a contagem diminui e, ao zerar, mostra "Reserva expirada" sem formulário', async () => {
    salvarReserva({ ...reserva, expiraEm: new Date(AGORA + 3000).toISOString() })
    renderizar()
    expect(screen.getByRole('timer')).toHaveTextContent('00:03')
    await avancar(1000)
    expect(screen.getByRole('timer')).toHaveTextContent('00:02')
    await avancar(2000)
    expect(screen.getByRole('heading', { name: 'Reserva expirada' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Pagar' })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Escolher assentos' })).toBeInTheDocument()
  })

  it('reserva já vencida ao abrir aparece como expirada', () => {
    salvarReserva({ ...reserva, expiraEm: new Date(AGORA - 1000).toISOString() })
    renderizar()
    expect(screen.getByRole('heading', { name: 'Reserva expirada' })).toBeInTheDocument()
  })

  it('expirar durante o processamento impede a aprovação', async () => {
    salvarReserva({ ...reserva, expiraEm: new Date(AGORA + 1500).toISOString() })
    renderizar()
    await preencher()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    expect(screen.queryByRole('heading', { name: 'Pagamento aprovado' })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Reserva expirada' })).toBeInTheDocument()
    expect(lerReserva('c1', 'f1', null)).not.toBeNull()
  })

  it('recarregar (nova montagem) com reserva válida mantém a página; após aprovado cai em "sem reserva"', async () => {
    salvarReserva(reserva)
    const { unmount } = render(<PaginaPagamento cinemaId="c1" filmeId="f1" sessao={login} irParaLogin={() => undefined} agora={() => relogio} />)
    unmount()
    renderizar()
    expect(screen.getByRole('heading', { name: 'Rocky' })).toBeInTheDocument()
    await preencher()
    await userEvent.click(screen.getByRole('button', { name: 'Pagar' }))
    await avancar(2000)
    document.body.innerHTML = ''
    renderizar()
    expect(screen.getByRole('heading', { name: 'Nenhuma reserva em andamento' })).toBeInTheDocument()
  })
})
