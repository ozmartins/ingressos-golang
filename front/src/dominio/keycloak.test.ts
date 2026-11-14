import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  concluirAutenticacao,
  idDoUsuario,
  lerSessao,
  retornoSeguro,
  salvarSessao,
  temRetorno,
  urlAutenticacao,
  urlLogout,
} from './keycloak'

afterEach(() => sessionStorage.clear())

describe('keycloak', () => {
  it('monta as URLs de login e registro com PKCE', async () => {
    const login = new URL(await urlAutenticacao('login', '/x'))
    expect(login.pathname).toBe('/realms/cinema/protocol/openid-connect/auth')
    expect(login.searchParams.get('code_challenge_method')).toBe('S256')
    const reg = new URL(await urlAutenticacao('registro', '/x'))
    expect(reg.pathname).toBe('/realms/cinema/protocol/openid-connect/registrations')
  })

  it('conclui a autenticação trocando o code pelo token', async () => {
    const url = new URL(await urlAutenticacao('login', '/cinemas/c1'))
    const estado = url.searchParams.get('state')!
    const fetchFn = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ access_token: 'a', id_token: 'i', expires_in: 60 })),
    )
    const r = await concluirAutenticacao(`?state=${estado}&code=abc`, fetchFn)
    expect(r?.sessao.accessToken).toBe('a')
    expect(r?.retorno).toBe('/cinemas/c1')
  })

  it('recusa state inválido', async () => {
    await urlAutenticacao('login', '/')
    const fetchFn = vi.fn()
    expect(await concluirAutenticacao('?state=outro&code=abc', fetchFn)).toBeNull()
    expect(fetchFn).not.toHaveBeenCalled()
  })

  it('só devolve sessão não expirada', () => {
    salvarSessao({ accessToken: 'a', idToken: null, expiraEm: 1000 })
    expect(lerSessao(500)?.accessToken).toBe('a')
    expect(lerSessao(2000)).toBeNull()
  })

  it('detecta retorno e monta o logout', () => {
    expect(temRetorno('?state=s&code=c')).toBe(true)
    expect(temRetorno('?aba=cinemas')).toBe(false)
    const u = new URL(urlLogout({ accessToken: 'a', idToken: 'i', expiraEm: 1 }))
    expect(u.searchParams.get('id_token_hint')).toBe('i')
  })
})

describe('retornoSeguro', () => {
  it('aceita caminhos internos e rejeita o resto', () => {
    expect(retornoSeguro('/cinemas/c1/filmes/f1/comprar?sessao=s')).toBe('/cinemas/c1/filmes/f1/comprar?sessao=s')
    expect(retornoSeguro('//evil.com')).toBe('/')
    expect(retornoSeguro('https://evil.com')).toBe('/')
    expect(retornoSeguro('/\\evil.com')).toBe('/')
  })
})

describe('idDoUsuario', () => {
  const jwt = (carga: object) => `h.${btoa(JSON.stringify(carga)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')}.s`

  it('lê o sub do JWT (base64url sem preenchimento)', () => {
    expect(idDoUsuario(jwt({ sub: 'abc-123' }))).toBe('abc-123')
  })

  it.each([null, '', 'tok', 'a.b.c', `x.${btoa('{"sub":42}')}.y`, `x.${btoa('{}')}.y`])('devolve null para %j', (token) => {
    expect(idDoUsuario(token)).toBeNull()
  })
})
