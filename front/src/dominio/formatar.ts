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

const moeda = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })
const hora = new Intl.DateTimeFormat('pt-BR', { hour: '2-digit', minute: '2-digit' })
const dia = new Intl.DateTimeFormat('pt-BR', { weekday: 'short', day: '2-digit', month: '2-digit' })
const dataHora = new Intl.DateTimeFormat('pt-BR', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

/** "42.00" → "R$ 42,00". */
export function formatarPreco(valor: string | number): string {
  return moeda.format(Number(valor))
}

/** Hora local: "19:30". */
export function formatarHora(data: Date): string {
  return hora.format(data)
}

/** Dia local `AAAA-MM-DD` → "sáb., 14/11". */
export function formatarDia(diaLocal: string): string {
  const [ano, mes, d] = diaLocal.split('-').map(Number)
  return dia.format(new Date(ano, mes - 1, d))
}

/** "14/11/2026 19:30" no fuso local. */
export function formatarDataHora(data: Date): string {
  return dataHora.format(data).replace(',', '')
}
