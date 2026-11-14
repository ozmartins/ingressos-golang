import { describe, expect, it } from 'vitest'
import { BASE_TESTE, fakeFetch, fetchQueFalha, respostaJson } from './fakes'
import { concluirLogin, desafioPkce, iniciarLogin, temRetorno, type Armazem, type ConfigOidc } from './oidc'

function armazemFake(): Armazem & { dados: Map<string, string> } {
  const dados = new Map<string, string>()
  return {
    dados,
    getItem: (k) => dados.get(k) ?? null,
    setItem: (k, v) => void dados.set(k, v),
    removeItem: (k) => void dados.delete(k),
  }
}

function config(): ConfigOidc & { armazem: ReturnType<typeof armazemFake> } {
  return { urlKeycloak: 'http://keycloak:8081', redirectUri: 'http://localhost:5173/', armazem: armazemFake() }
}

describe('desafioPkce', () => {
  it('calcula S256 conforme o exemplo da RFC 7636', async () => {
    expect(await desafioPkce('dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk')).toBe(
      'E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM',
    )
  })
})

describe('iniciarLogin', () => {
  it('monta a URL da página de login com PKCE e guarda state/verifier', async () => {
    const c = config()
    const url = new URL(await iniciarLogin(c))
    expect(url.origin + url.pathname).toBe('http://keycloak:8081/realms/cinema/protocol/openid-connect/auth')
    expect(url.searchParams.get('client_id')).toBe('cinema-app')
    expect(url.searchParams.get('response_type')).toBe('code')
    expect(url.searchParams.get('redirect_uri')).toBe('http://localhost:5173/')
    expect(url.searchParams.get('code_challenge_method')).toBe('S256')
    expect(url.searchParams.get('state')).toBe(c.armazem.dados.get('oidc:estado'))
    const verificador = c.armazem.dados.get('oidc:verificador')!
    expect(url.searchParams.get('code_challenge')).toBe(await desafioPkce(verificador))
  })
})

describe('temRetorno', () => {
  it('detecta code+state ou error+state', () => {
    expect(temRetorno('?code=a&state=b')).toBe(true)
    expect(temRetorno('?error=access_denied&state=b')).toBe(true)
    expect(temRetorno('?code=a')).toBe(false)
    expect(temRetorno('')).toBe(false)
  })
})

describe('concluirLogin', () => {
  async function preparado() {
    const c = config()
    await iniciarLogin(c)
    return { c, estado: c.armazem.dados.get('oidc:estado')!, verificador: c.armazem.dados.get('oidc:verificador')! }
  }

  it('troca o code pelo token e limpa state/verifier', async () => {
    const { c, estado, verificador } = await preparado()
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { access_token: 'tok' }))
    const r = await concluirLogin(fn, BASE_TESTE, c, `?code=abc&state=${estado}`)
    expect(r.sessao).toEqual({ token: 'tok' })
    expect(chamadas[0].url).toBe('/identidade/realms/cinema/protocol/openid-connect/token')
    const corpo = new URLSearchParams(String(chamadas[0].init.body))
    expect(corpo.get('grant_type')).toBe('authorization_code')
    expect(corpo.get('code')).toBe('abc')
    expect(corpo.get('code_verifier')).toBe(verificador)
    expect(corpo.get('redirect_uri')).toBe('http://localhost:5173/')
    expect(c.armazem.dados.size).toBe(0)
    expect(r.linhas.map((l) => l.texto).join('')).not.toContain('tok')
  })

  it('state diferente é recusado sem chamar o Keycloak', async () => {
    const { c } = await preparado()
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { access_token: 'tok' }))
    const r = await concluirLogin(fn, BASE_TESTE, c, '?code=abc&state=forjado')
    expect(r.sessao).toBeUndefined()
    expect(chamadas).toHaveLength(0)
    expect(r.linhas[0].tipo).toBe('erro')
  })

  it('sem state guardado é recusado', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const r = await concluirLogin(fn, BASE_TESTE, config(), '?code=abc&state=x')
    expect(r.sessao).toBeUndefined()
    expect(chamadas).toHaveLength(0)
  })

  it('erro devolvido pelo Keycloak vira mensagem', async () => {
    const { c, estado } = await preparado()
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    const r = await concluirLogin(fn, BASE_TESTE, c, `?error=access_denied&state=${estado}`)
    expect(r.sessao).toBeUndefined()
    expect(r.linhas[0].texto).toContain('access_denied')
  })

  it('troca recusada (400) não gera sessão', async () => {
    const { c, estado } = await preparado()
    const { fn } = fakeFetch(() => respostaJson(400, { error: 'invalid_grant' }))
    const r = await concluirLogin(fn, BASE_TESTE, c, `?code=abc&state=${estado}`)
    expect(r.sessao).toBeUndefined()
    expect(r.linhas[0].texto).toContain('400')
  })

  it('falha de rede vira mensagem', async () => {
    const { c, estado } = await preparado()
    const r = await concluirLogin(fetchQueFalha(), BASE_TESTE, c, `?code=abc&state=${estado}`)
    expect(r.sessao).toBeUndefined()
    expect(r.linhas[0].tipo).toBe('erro')
  })
})
