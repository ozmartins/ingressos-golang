// Autenticação pelas telas nativas do Keycloak: Authorization Code + PKCE (S256).
// A senha é digitada só no Keycloak; o front recebe apenas o code e o troca por tokens.

const CLIENT_ID = 'cinema-app'
const BASE_PADRAO = 'http://keycloak:8081'
const CHAVE_AUTH = 'oidc:auth'
const CHAVE_SESSAO = 'oidc:sessao'

export interface Sessao {
  accessToken: string
  idToken: string | null
  /** Instante (ms desde a época) em que o access token expira. */
  expiraEm: number
}

export type Tela = 'login' | 'registro'

interface AuthPendente {
  estado: string
  verificador: string
  retorno: string
}

function base(): string {
  return (import.meta.env.VITE_KEYCLOAK_URL ?? BASE_PADRAO).replace(/\/+$/, '')
}

const oidc = (parte: string) => `${base()}/realms/cinema/protocol/openid-connect/${parte}`

/** Para onde o Keycloak devolve o usuário após o login/registro/logout. */
export function urlDeRetorno(): string {
  return `${window.location.origin}/`
}

function base64Url(bytes: Uint8Array): string {
  let binario = ''
  for (const b of bytes) binario += String.fromCharCode(b)
  return btoa(binario).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function aleatorio(): string {
  return base64Url(crypto.getRandomValues(new Uint8Array(32)))
}

async function desafio(verificador: string): Promise<string> {
  const resumo = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verificador))
  return base64Url(new Uint8Array(resumo))
}

/** Guarda state/verifier/página de origem e devolve a URL da tela do Keycloak. */
export async function urlAutenticacao(tela: Tela, retorno: string): Promise<string> {
  const estado = aleatorio()
  const verificador = aleatorio()
  const pendente: AuthPendente = { estado, verificador, retorno }
  sessionStorage.setItem(CHAVE_AUTH, JSON.stringify(pendente))
  const consulta = new URLSearchParams({
    client_id: CLIENT_ID,
    response_type: 'code',
    scope: 'openid',
    redirect_uri: urlDeRetorno(),
    state: estado,
    code_challenge: await desafio(verificador),
    code_challenge_method: 'S256',
  })
  return `${oidc(tela === 'login' ? 'auth' : 'registrations')}?${consulta.toString()}`
}

/** Há retorno do Keycloak na URL (?code=... ou ?error=...)? */
export function temRetorno(search: string): boolean {
  const p = new URLSearchParams(search)
  return p.has('state') && (p.has('code') || p.has('error'))
}

function lerPendente(): AuthPendente | null {
  try {
    const bruto = sessionStorage.getItem(CHAVE_AUTH)
    return bruto === null ? null : (JSON.parse(bruto) as AuthPendente)
  } catch {
    return null
  }
}

/** Troca o code por tokens. Devolve a sessão e a página para onde voltar; `null` se recusado. */
export async function concluirAutenticacao(
  search: string,
  fetchFn: typeof fetch = fetch.bind(globalThis),
): Promise<{ sessao: Sessao; retorno: string } | null> {
  const pendente = lerPendente()
  sessionStorage.removeItem(CHAVE_AUTH)
  const p = new URLSearchParams(search)
  const code = p.get('code')
  if (pendente === null || p.get('state') !== pendente.estado || code === null || code === '') return null

  const resposta = await fetchFn(oidc('token'), {
    method: 'POST',
    headers: { Accept: 'application/json', 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      client_id: CLIENT_ID,
      code,
      redirect_uri: urlDeRetorno(),
      code_verifier: pendente.verificador,
    }).toString(),
  })
  if (!resposta.ok) return null
  const dados = (await resposta.json()) as {
    access_token?: unknown
    id_token?: unknown
    expires_in?: unknown
  }
  if (typeof dados.access_token !== 'string' || dados.access_token === '') return null
  const sessao: Sessao = {
    accessToken: dados.access_token,
    idToken: typeof dados.id_token === 'string' ? dados.id_token : null,
    expiraEm: Date.now() + (typeof dados.expires_in === 'number' ? dados.expires_in : 300) * 1000,
  }
  return { sessao, retorno: pendente.retorno }
}

export function salvarSessao(sessao: Sessao): void {
  sessionStorage.setItem(CHAVE_SESSAO, JSON.stringify(sessao))
}

export function limparSessao(): void {
  sessionStorage.removeItem(CHAVE_SESSAO)
}

/** Sessão guardada ainda dentro da validade, ou `null`. */
export function lerSessao(agora: number = Date.now()): Sessao | null {
  try {
    const bruto = sessionStorage.getItem(CHAVE_SESSAO)
    if (bruto === null) return null
    const s = JSON.parse(bruto) as Sessao
    return typeof s.accessToken === 'string' && s.expiraEm > agora ? s : null
  } catch {
    return null
  }
}

/** URL que encerra a sessão no Keycloak e volta ao app. */
export function urlLogout(sessao: Sessao): string {
  const consulta = new URLSearchParams({
    client_id: CLIENT_ID,
    post_logout_redirect_uri: urlDeRetorno(),
  })
  if (sessao.idToken !== null) consulta.set('id_token_hint', sessao.idToken)
  return `${oidc('logout')}?${consulta.toString()}`
}
