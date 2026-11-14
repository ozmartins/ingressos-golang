/** Instantâneo da reserva criada em "Continuar", lido pela página de pagamento (nunca guarda dados de cartão). */
export interface ReservaPagamento {
  /** Dono da reserva (`sub` do login); outra pessoa na mesma aba não a enxerga. */
  usuarioId: string | null
  cinemaId: string
  filmeId: string
  sessaoId: string
  filmeTitulo: string
  posterUrl: string | null
  cinemaNome: string
  sala: number
  /** ISO 8601 do início da sessão. */
  inicio: string
  rotulos: string[]
  precoCentavos: number
  taxaCentavos: number
  totalCentavos: number
  /** ISO 8601 do fim do prazo da reserva. */
  expiraEm: string
}

const CHAVE = 'pagamento:reserva'

export function salvarReserva(reserva: ReservaPagamento): void {
  try {
    sessionStorage.setItem(CHAVE, JSON.stringify(reserva))
  } catch {
    // sem armazenamento disponível: a página de pagamento mostrará "sem reserva"
  }
}

export function limparReserva(): void {
  try {
    sessionStorage.removeItem(CHAVE)
  } catch {
    // nada a limpar
  }
}

const texto = (v: unknown): v is string => typeof v === 'string' && v !== ''
const inteiro = (v: unknown): v is number => typeof v === 'number' && Number.isInteger(v) && v >= 0

/** Reserva salva para este cinema/filme e usuário, ou `null` se ausente, inválida ou de outro usuário. */
export function lerReserva(cinemaId: string, filmeId: string, usuarioId: string | null): ReservaPagamento | null {
  try {
    const bruto = sessionStorage.getItem(CHAVE)
    if (bruto === null) return null
    const r = JSON.parse(bruto) as Partial<ReservaPagamento> | null
    if (typeof r !== 'object' || r === null) return null
    const valida =
      r.usuarioId === usuarioId &&
      r.cinemaId === cinemaId &&
      r.filmeId === filmeId &&
      texto(r.sessaoId) &&
      texto(r.filmeTitulo) &&
      (r.posterUrl === null || typeof r.posterUrl === 'string') &&
      texto(r.cinemaNome) &&
      inteiro(r.sala) &&
      texto(r.inicio) &&
      Array.isArray(r.rotulos) &&
      r.rotulos.length > 0 &&
      r.rotulos.every(texto) &&
      inteiro(r.precoCentavos) &&
      inteiro(r.taxaCentavos) &&
      inteiro(r.totalCentavos) &&
      texto(r.expiraEm) &&
      !Number.isNaN(new Date(r.expiraEm).getTime())
    return valida ? (r as ReservaPagamento) : null
  } catch {
    return null
  }
}

/** Segundos até o fim do prazo (mínimo 0). */
export function segundosRestantes(expiraEm: string, agora: number = Date.now()): number {
  return Math.max(0, Math.ceil((new Date(expiraEm).getTime() - agora) / 1000))
}

/** 125 → "02:05". */
export function formatarContagem(segundos: number): string {
  const m = Math.floor(segundos / 60)
  const s = segundos % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
