import { describe, expect, it } from 'vitest'
import { BASE_TESTE, fakeFetch, fetchQueFalha, respostaJson } from './fakes'
import { SEM_SESSAO, login, logout, mascarar } from './sessao'

describe('login', () => {
  it('envia client_credentials form-urlencoded ao realm cinema e guarda o token', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { access_token: 'tok123' }))
    const r = await login(fn, BASE_TESTE, 'meu-client', 'meu-segredo')
    expect(r.sessao).toEqual({ token: 'tok123' })
    expect(chamadas[0].url).toBe('/identidade/realms/cinema/protocol/openid-connect/token')
    expect(chamadas[0].init.method).toBe('POST')
    expect(chamadas[0].init.headers).toHaveProperty('Content-Type', 'application/x-www-form-urlencoded')
    const corpo = new URLSearchParams(String(chamadas[0].init.body))
    expect(corpo.get('grant_type')).toBe('client_credentials')
    expect(corpo.get('client_id')).toBe('meu-client')
    expect(corpo.get('client_secret')).toBe('meu-segredo')
  })

  it('não mostra o token nem o segredo na saída', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, { access_token: 'tok123' }))
    const r = await login(fn, BASE_TESTE, 'meu-client', 'meu-segredo')
    const saida = r.linhas.map((l) => l.texto).join('\n')
    expect(saida).not.toContain('tok123')
    expect(saida).not.toContain('meu-segredo')
  })

  it('credencial inválida não gera sessão e não ecoa o segredo', async () => {
    const { fn } = fakeFetch(() => respostaJson(401, { error: 'unauthorized_client' }))
    const r = await login(fn, BASE_TESTE, 'c', 'segredo-errado')
    expect(r.sessao).toBeUndefined()
    const saida = r.linhas.map((l) => l.texto).join('\n')
    expect(saida).toContain('falha na autenticação (401)')
    expect(saida).not.toContain('segredo-errado')
  })

  it('resposta 200 sem access_token não gera sessão', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, { outro: 1 }))
    expect((await login(fn, BASE_TESTE, 'c', 's')).sessao).toBeUndefined()
  })

  it('falha de rede vira mensagem', async () => {
    const r = await login(fetchQueFalha(), BASE_TESTE, 'c', 's')
    expect(r.sessao).toBeUndefined()
    expect(r.linhas[0].tipo).toBe('erro')
  })
})

describe('logout e mascarar', () => {
  it('logout descarta o token', () => {
    expect(logout().sessao).toEqual(SEM_SESSAO)
  })

  it('mascarar oculta o segredo do login', () => {
    expect(mascarar('login meu-client meu-segredo')).toBe('login meu-client ****')
    expect(mascarar('  LOGIN  c   s3gr3d0 ')).toBe('  LOGIN  c   ****')
  })

  it('mascarar não altera outras entradas', () => {
    expect(mascarar('echo login a b')).toBe('echo login a b')
    expect(mascarar('login')).toBe('login')
    expect(mascarar('login so-o-id')).toBe('login so-o-id')
    expect(mascarar('filmes listar')).toBe('filmes listar')
  })
})
