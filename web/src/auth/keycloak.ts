// O emissor dos tokens é `http://keycloak:8081/realms/cinema`, endereço interno
// do compose. O navegador chega nele pelo `/auth/` do nginx, e o token sai com
// o emissor intacto — é assim que os quatro serviços continuam validando sem
// nenhuma mudança de configuração.
//
// O grant é o de senha (Direct Access Grant), já habilitado no realm. Não é o
// que se usaria em produção; é o que dispensa mexer no emissor, que hoje o
// catálogo lê da mesma variável que o JWKS.

const ROTA_TOKEN = '/auth/realms/cinema/protocol/openid-connect/token'
const ROTA_LOGOUT = '/auth/realms/cinema/protocol/openid-connect/logout'
const CLIENTE = 'cinema-app'

export interface Sessao {
  accessToken: string
  refreshToken: string
  /** Instante (ms) em que o access token expira, já com a margem aplicada. */
  expiraEm: number
  usuario: string
  sub: string
}

export class ErroDeLogin extends Error {}

interface RespostaDeToken {
  access_token: string
  refresh_token: string
  expires_in: number
}

/** Lê o payload do JWT sem verificar assinatura: aqui ele serve só para
 *  exibir o nome e guardar o `sub`. Quem valida de verdade são os serviços. */
function lerPayload(token: string): Record<string, unknown> {
  const meio = token.split('.')[1]
  if (!meio) return {}
  const base64 = meio.replace(/-/g, '+').replace(/_/g, '/')
  const texto = decodeURIComponent(
    atob(base64)
      .split('')
      .map((c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0'))
      .join(''),
  )
  return JSON.parse(texto) as Record<string, unknown>
}

function montarSessao(resposta: RespostaDeToken): Sessao {
  const payload = lerPayload(resposta.access_token)
  return {
    accessToken: resposta.access_token,
    refreshToken: resposta.refresh_token,
    // Trinta segundos de folga para que nenhuma requisição saia com um token
    // que expira no caminho.
    expiraEm: Date.now() + (resposta.expires_in - 30) * 1000,
    usuario: String(payload.preferred_username ?? payload.name ?? 'usuário'),
    sub: String(payload.sub ?? ''),
  }
}

async function pedirToken(corpo: URLSearchParams): Promise<Sessao> {
  let resposta: Response
  try {
    resposta = await fetch(ROTA_TOKEN, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: corpo,
    })
  } catch {
    throw new ErroDeLogin('Não foi possível falar com o Keycloak.')
  }
  if (!resposta.ok) {
    const detalhe = await resposta.json().catch(() => ({}))
    const descricao = (detalhe as Record<string, string>).error_description
    throw new ErroDeLogin(
      resposta.status === 401 ? 'Usuário ou senha inválidos.' : descricao || 'Falha ao autenticar.',
    )
  }
  return montarSessao((await resposta.json()) as RespostaDeToken)
}

export const entrar = (usuario: string, senha: string) =>
  pedirToken(
    new URLSearchParams({
      grant_type: 'password',
      client_id: CLIENTE,
      username: usuario,
      password: senha,
      scope: 'openid',
    }),
  )

export const renovar = (refreshToken: string) =>
  pedirToken(new URLSearchParams({ grant_type: 'refresh_token', client_id: CLIENTE, refresh_token: refreshToken }))

/** Invalida o refresh token no Keycloak. Falha em silêncio: sair da sessão
 *  local não pode depender do servidor responder. */
export async function sair(refreshToken: string): Promise<void> {
  try {
    await fetch(ROTA_LOGOUT, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ client_id: CLIENTE, refresh_token: refreshToken }),
    })
  } catch {
    /* sem efeito: a sessão local some de qualquer modo */
  }
}
