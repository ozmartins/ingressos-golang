import { chamar } from './cliente'
import type { ReservaConfirmada, RespostaMapa } from './tipos'

/** Leitura do mapa de poltronas. */
export const consultarMapa = (token: string, sessaoId: string) =>
  chamar<RespostaMapa>('estoque', `/sessoes/${sessaoId}/poltronas`, { token })

/** O caminho de compra: o estoque consulta o preço e o status da sessão no
 *  catálogo e calcula o valor. O corpo leva só as poltronas. */
export const reservarPoltronas = (token: string, sessaoId: string, poltronas: string[]) =>
  chamar<ReservaConfirmada>('estoque', `/sessoes/${sessaoId}/bloqueios`, {
    metodo: 'POST',
    corpo: { poltronas_ids: poltronas },
    token,
  })
