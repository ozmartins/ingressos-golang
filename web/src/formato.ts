/** `preco_base` e afins chegam como texto decimal exato. Formatar sem passar
 *  por `Number` mantém o centavo intacto — este é o valor que vira cobrança. */
export function moedaDeTexto(valor: string): string {
  const [inteiro, decimais = ''] = valor.split('.')
  const centavos = (decimais + '00').slice(0, 2)
  const comSeparador = inteiro.replace(/\B(?=(\d{3})+(?!\d))/g, '.')
  return `R$ ${comSeparador},${centavos}`
}

/** O pagamento devolve `valor_total` como número, e aí não há o que preservar. */
export function moedaDeNumero(valor: number): string {
  return valor.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })
}

const dataHora = new Intl.DateTimeFormat('pt-BR', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

export const formatarDataHora = (iso: string): string => dataHora.format(new Date(iso))

export const formatarHora = (iso: string): string =>
  new Date(iso).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })

/** O input `datetime-local` não aceita offset; o serviço quer RFC 3339. */
export const paraIsoUtc = (localDoInput: string): string => new Date(localDoInput).toISOString()

export function paraInputLocal(iso: string): string {
  const d = new Date(iso)
  const ajustada = new Date(d.getTime() - d.getTimezoneOffset() * 60000)
  return ajustada.toISOString().slice(0, 16)
}
