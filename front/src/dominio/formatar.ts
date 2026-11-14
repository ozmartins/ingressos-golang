import type { StatusFilme } from '../api/tipos'

/** 125 → "2h 05min"; 90 → "1h 30min"; 45 → "45min". */
export function formatarDuracao(minutos: number): string {
  const horas = Math.floor(minutos / 60)
  const resto = minutos % 60
  if (horas === 0) return `${resto}min`
  return `${horas}h ${String(resto).padStart(2, '0')}min`
}

export const ROTULO_STATUS: Record<StatusFilme, string> = {
  EM_CARTAZ: 'Em cartaz',
  BREVE: 'Em breve',
  FORA_DE_CARTAZ: 'Fora de cartaz',
}
