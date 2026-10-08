import { chamar } from './cliente'
import type { Ingresso, ResultadoValidacao, StatusIngresso } from './tipos'

export const listarMeusIngressos = (token: string, status?: StatusIngresso) =>
  chamar<Ingresso[]>('notificacao', '/ingressos/meus-ingressos', { token, consulta: { status } })

/** A portaria não usa JWT: autentica pela chave do dispositivo, que o operador
 *  informa na tela e não vem no bundle. */
export const validarIngresso = (chaveDeApi: string, codigoQr: string) =>
  chamar<ResultadoValidacao>('notificacao', '/ingressos/validar', {
    metodo: 'POST',
    corpo: { codigo_qr: codigoQr },
    chaveDeApi,
  })
