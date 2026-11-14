import { afterEach, describe, expect, it } from 'vitest'
import { formatarContagem, lerReserva, limparReserva, salvarReserva, segundosRestantes, type ReservaPagamento } from './reservaPagamento'

const base: ReservaPagamento = {
  usuarioId: 'u1',
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
  expiraEm: '2026-11-15T21:10:00.000Z',
}

afterEach(() => sessionStorage.clear())

describe('reservaPagamento', () => {
  it('salva e lê a reserva do cinema/filme', () => {
    salvarReserva(base)
    expect(lerReserva('c1', 'f1', 'u1')).toEqual(base)
  })

  it('aceita reserva sem classificação (gravada antes) e com ela; rejeita tipo inválido', () => {
    salvarReserva({ ...base, classificacao: '12' })
    expect(lerReserva('c1', 'f1', 'u1')?.classificacao).toBe('12')
    salvarReserva(base)
    expect(lerReserva('c1', 'f1', 'u1')?.classificacao).toBeUndefined()
    sessionStorage.setItem('pagamento:reserva', JSON.stringify({ ...base, classificacao: 12 }))
    expect(lerReserva('c1', 'f1', 'u1')).toBeNull()
  })

  it('devolve null sem reserva, com ids diferentes ou após limpar', () => {
    expect(lerReserva('c1', 'f1', 'u1')).toBeNull()
    salvarReserva(base)
    expect(lerReserva('c2', 'f1', 'u1')).toBeNull()
    expect(lerReserva('c1', 'f2', 'u1')).toBeNull()
    limparReserva()
    expect(lerReserva('c1', 'f1', 'u1')).toBeNull()
  })

  it('não entrega a reserva a outro usuário (nem a sessão sem identificação)', () => {
    salvarReserva(base)
    expect(lerReserva('c1', 'f1', 'u2')).toBeNull()
    expect(lerReserva('c1', 'f1', null)).toBeNull()
    salvarReserva({ ...base, usuarioId: null })
    expect(lerReserva('c1', 'f1', null)).not.toBeNull()
    expect(lerReserva('c1', 'f1', 'u1')).toBeNull()
  })

  it.each([
    ['JSON malformado', '{x'],
    ['valor não objeto', '42'],
    ['null', 'null'],
    ['rótulos vazios', JSON.stringify({ ...base, rotulos: [] })],
    ['campo faltando', JSON.stringify({ ...base, sessaoId: undefined })],
    ['total negativo', JSON.stringify({ ...base, totalCentavos: -1 })],
    ['prazo inválido', JSON.stringify({ ...base, expiraEm: 'ontem' })],
  ])('rejeita %s', (_nome, bruto) => {
    sessionStorage.setItem('pagamento:reserva', bruto)
    expect(lerReserva('c1', 'f1', 'u1')).toBeNull()
  })

  it('calcula segundos restantes e formata a contagem', () => {
    const agora = new Date('2026-11-15T21:00:00.000Z').getTime()
    expect(segundosRestantes(base.expiraEm, agora)).toBe(600)
    expect(segundosRestantes(base.expiraEm, agora + 700_000)).toBe(0)
    expect(formatarContagem(125)).toBe('02:05')
    expect(formatarContagem(0)).toBe('00:00')
  })
})
