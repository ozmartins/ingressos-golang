import { chamar } from './cliente'
import type { FormaPagamento, Pagamento } from './tipos'

/** A transação nasce do consumo de `reserva.criada`, então `404` logo depois
 *  da reserva significa "ainda não chegou", e não "não existe". Quem chama
 *  precisa tolerar isso por alguns segundos. */
export const consultarPagamento = (token: string, reservaId: string) =>
  chamar<Pagamento>('pagamento', `/pagamentos/reserva/${reservaId}`, { token })

/** Responde 202: a cobrança acontece fora desta requisição, e o desfecho chega
 *  pelo GET. */
export const escolherForma = (token: string, reservaId: string, forma: FormaPagamento) =>
  chamar<void>('pagamento', `/pagamentos/reserva/${reservaId}`, {
    metodo: 'POST',
    corpo: { forma_pagamento: forma },
    token,
  })
