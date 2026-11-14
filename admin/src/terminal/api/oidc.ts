import type { Linha } from '../linhas'
import { FalhaRequisicao, chamar, type Base } from './cliente'
import { formatarFalha } from './formatar'
import type { Sessao } from './sessao'

// Login pela página nativa do Keycloak: Authorization Code + PKCE (S256).
// A senha é digitada só na página do Keycloak; o front recebe apenas o code.

export const CLIENT_ID = 'cinema-app'
const CHAVE_ESTADO = 'oidc:estado'
const CHAVE_VERIFICADOR = 'oidc:verificador'

export interface Armazem {
  getItem(chave: string): string | null
  setItem(chave: string, valor: string): void
  removeItem(chave: string): void
}

export interface ConfigOidc {
  /** Endereço do Keycloak alcançável pelo navegador (para a tela de login). */
  urlKeycloak: string
  /** Para onde o Keycloak devolve o usuário após o login. */
  redirectUri: string
  armazem: Armazem
}

export interface ResultadoConclusao {
  linhas: Linha[]
  sessao?: Sessao
}

function base64Url(bytes: Uint8Array): string {
  let binario = ''
  for (const b of bytes) binario += String.fromCharCode(b)
  return btoa(binario).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function aleatorio(): string {
  return base64Url(crypto.getRandomValues(new Uint8Array(32)))
}

export async function desafioPkce(verificador: string): Promise<string> {
  const resumo = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verificador))
  return base64Url(new Uint8Array(resumo))
}

/** Guarda state/verifier e devolve a URL da página de login do Keycloak. */
export async function iniciarLogin(config: ConfigOidc): Promise<string> {
  const estado = aleatorio()
  const verificador = aleatorio()
  config.armazem.setItem(CHAVE_ESTADO, estado)
  config.armazem.setItem(CHAVE_VERIFICADOR, verificador)
  const consulta = new URLSearchParams({
    client_id: CLIENT_ID,
    response_type: 'code',
    scope: 'openid',
    redirect_uri: config.redirectUri,
    state: estado,
    code_challenge: await desafioPkce(verificador),
    code_challenge_method: 'S256',
  })
  return `${config.urlKeycloak}/realms/cinema/protocol/openid-connect/auth?${consulta.toString()}`
}

/** Há retorno do Keycloak na URL (?code=... ou ?error=...)? */
export function temRetorno(search: string): boolean {
  const p = new URLSearchParams(search)
  return p.has('state') && (p.has('code') || p.has('error'))
}

const erro = (texto: string): Linha => ({ tipo: 'erro', texto })

/** Troca o code recebido por um access_token. Consome state/verifier. */
export async function concluirLogin(
  fetchFn: typeof fetch,
  base: Base,
  config: ConfigOidc,
  search: string,
): Promise<ResultadoConclusao> {
  const p = new URLSearchParams(search)
  const estadoEsperado = config.armazem.getItem(CHAVE_ESTADO)
  const verificador = config.armazem.getItem(CHAVE_VERIFICADOR)
  config.armazem.removeItem(CHAVE_ESTADO)
  config.armazem.removeItem(CHAVE_VERIFICADOR)

  if (estadoEsperado === null || verificador === null || p.get('state') !== estadoEsperado) {
    return { linhas: [erro('login recusado: estado inválido. Digite login para tentar de novo.')] }
  }
  const recusa = p.get('error')
  if (recusa !== null) {
    return { linhas: [erro(`login não concluído: ${p.get('error_description') ?? recusa}`)] }
  }
  const code = p.get('code')
  if (code === null || code === '') {
    return { linhas: [erro('login recusado: resposta sem code.')] }
  }

  const corpo = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: CLIENT_ID,
    code,
    redirect_uri: config.redirectUri,
    code_verifier: verificador,
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
    const token = resposta.status >= 200 && resposta.status < 300 ? extrairToken(resposta.texto) : null
    if (token === null) {
      return { linhas: [erro(`falha ao obter o token (${resposta.status}). Digite login para tentar de novo.`)] }
    }
    return {
      linhas: [{ tipo: 'sistema', texto: 'Autenticado. Comandos de escrita liberados.' }],
      sessao: { token },
    }
  } catch (e) {
    if (e instanceof FalhaRequisicao) return { linhas: formatarFalha(e) }
    throw e
  }
}

function extrairToken(texto: string): string | null {
  try {
    const dados = JSON.parse(texto) as { access_token?: unknown }
    return typeof dados.access_token === 'string' && dados.access_token !== ''
      ? dados.access_token
      : null
  } catch {
    return null
  }
}
