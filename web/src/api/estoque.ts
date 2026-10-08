import { chamar } from './cliente'
import type { RespostaMapa } from './tipos'

/** Leitura do mapa de poltronas. A escrita passa pelo catálogo, que é quem
 *  tem autoridade sobre o preço da sessão. */
export const consultarMapa = (token: string, sessaoId: string) =>
  chamar<RespostaMapa>('estoque', `/sessoes/${sessaoId}/poltronas`, { token })
