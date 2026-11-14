import type { Linha } from '../linhas'
import { FalhaRequisicao, chamar, type Base } from './cliente'
import { formatarFalha } from './formatar'

export interface Sessao {
  token: string | null
}

export const SEM_SESSAO: Sessao = { token: null }

export interface ResultadoLogin {
  linhas: Linha[]
  sessao?: Sessao
}

export async function login(
  fetchFn: typeof fetch,
  base: Base,
  clientId: string,
  clientSecret: string,
): Promise<ResultadoLogin> {
  const corpo = new URLSearchParams({
    grant_type: 'client_credentials',
    client_id: clientId,
    client_secret: clientSecret,
  })
  try {
    const resposta = await chamar(
      fetchFn,
      `${base.identidade}/realms/cinema/protocol/openid-connect/token`,
      {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: corpo.toString(),
      },
    )
    if (resposta.status >= 200 && resposta.status < 300) {
      const token = extrairToken(resposta.texto)
      if (token !== null) {
        return {
          linhas: [{ tipo: 'sistema', texto: 'Autenticado. Comandos de escrita liberados.' }],
          sessao: { token },
        }
      }
    }
    return {
      linhas: [
        { tipo: 'erro', texto: `falha na autenticação (${resposta.status}): verifique as credenciais` },
      ],
    }
  } catch (e) {
    if (e instanceof FalhaRequisicao) return { linhas: formatarFalha(e) }
    throw e
  }
}

function extrairToken(texto: string): string | null {
  try {
    const dados = JSON.parse(texto) as unknown
    if (typeof dados === 'object' && dados !== null && 'access_token' in dados) {
      const token = (dados as { access_token: unknown }).access_token
      return typeof token === 'string' && token !== '' ? token : null
    }
  } catch {
    return null
  }
  return null
}

export function logout(): ResultadoLogin {
  return {
    linhas: [{ tipo: 'sistema', texto: 'Sessão encerrada.' }],
    sessao: SEM_SESSAO,
  }
}

export function mascarar(entrada: string): string {
  return entrada.replace(/^(\s*login\s+\S+\s+).+$/i, '$1****')
}
