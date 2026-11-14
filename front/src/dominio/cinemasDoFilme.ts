import type { Cinema, SessaoNaGrade } from '../api/tipos'

export interface CinemaDoFilme {
  cinema: Cinema
  quantidadeSessoes: number
}

/**
 * Cinemas ativos com ao menos uma sessão do filme com início futuro (as sessões
 * recebidas são de um único filme). Ordena por UF, cidade e nome.
 */
export function cinemasComOFilme(
  sessoes: SessaoNaGrade[],
  cinemas: Cinema[],
  agora: Date = new Date(),
): CinemaDoFilme[] {
  const sessoesPorCinema = new Map<string, number>()
  for (const sessao of sessoes) {
    if (new Date(sessao.data_hora_inicio).getTime() <= agora.getTime()) continue
    sessoesPorCinema.set(sessao.cinema_id, (sessoesPorCinema.get(sessao.cinema_id) ?? 0) + 1)
  }
  const resultado: CinemaDoFilme[] = []
  for (const cinema of cinemas) {
    const quantidadeSessoes = sessoesPorCinema.get(cinema.id)
    if (quantidadeSessoes) resultado.push({ cinema, quantidadeSessoes })
  }
  const ptBr = (a: string, b: string) => a.localeCompare(b, 'pt-BR')
  return resultado.sort(
    (a, b) =>
      ptBr(a.cinema.estado, b.cinema.estado) ||
      ptBr(a.cinema.cidade, b.cinema.cidade) ||
      ptBr(a.cinema.nome, b.cinema.nome),
  )
}
