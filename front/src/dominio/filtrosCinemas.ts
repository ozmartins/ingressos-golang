import type { Cinema } from '../api/tipos'
import { normalizar, ordenarPtBr } from './texto'

export interface FiltrosCinemas {
  texto: string
  estado: string
  cidade: string
}

export const filtrosCinemasVazios: FiltrosCinemas = { texto: '', estado: '', cidade: '' }

export function filtrarCinemas(cinemas: Cinema[], filtros: FiltrosCinemas): Cinema[] {
  const busca = normalizar(filtros.texto)
  return cinemas.filter(
    (cinema) =>
      (busca === '' || normalizar(cinema.nome).includes(busca)) &&
      (filtros.estado === '' || cinema.estado === filtros.estado) &&
      (filtros.cidade === '' || cinema.cidade === filtros.cidade),
  )
}

/** Estados existentes e cidades do estado escolhido (ou todas, sem estado). */
export function opcoesCinemas(cinemas: Cinema[], estado: string): { estados: string[]; cidades: string[] } {
  const doEstado = estado === '' ? cinemas : cinemas.filter((cinema) => cinema.estado === estado)
  return {
    estados: ordenarPtBr(cinemas.map((cinema) => cinema.estado)),
    cidades: ordenarPtBr(doEstado.map((cinema) => cinema.cidade)),
  }
}

/** Zera a cidade quando ela não pertence ao estado escolhido. */
export function ajustarCidade(cinemas: Cinema[], filtros: FiltrosCinemas): FiltrosCinemas {
  if (filtros.cidade === '') return filtros
  const { cidades } = opcoesCinemas(cinemas, filtros.estado)
  return cidades.includes(filtros.cidade) ? filtros : { ...filtros, cidade: '' }
}
