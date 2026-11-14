import { describe, expect, it } from 'vitest'
import { BASE_TESTE, fakeFetch, fetchQueFalha, respostaJson } from './api/fakes'
import { SEM_SESSAO, type Sessao } from './api/sessao'
import { despachar, type Contexto } from './despachar'

function ctx(fetchFn: typeof fetch, sessao: Sessao = SEM_SESSAO): Contexto {
  return { fetchFn, base: BASE_TESTE, sessao }
}
const comToken: Sessao = { token: 'tok' }
const textos = (r: { linhas: { texto: string }[] }) => r.linhas.map((l) => l.texto)

describe('despachar - leituras', () => {
  it('filmes listar aplica filtros e paginação', async () => {
    const { fn, chamadas } = fakeFetch(() =>
      respostaJson(200, {
        itens: [{ id: 'f1', titulo: 'The Matrix', duracao_minutos: 130, classificacao_etaria: '14', genero: 'Ação', status: 'EM_CARTAZ' }],
        pagina: { pagina: 2, tamanho: 5, total: 6, tem_proxima: false },
      }),
    )
    const r = await despachar('filmes listar status=EM_CARTAZ page=2 page_size=5', ctx(fn))
    expect(chamadas[0].url).toBe('/catalogo/api/v1/filmes?status=EM_CARTAZ&page=2&page_size=5')
    expect(chamadas[0].init.method).toBe('GET')
    expect(r.limpar).toBe(false)
    expect(textos(r).join('\n')).toContain('The Matrix')
    expect(textos(r).join('\n')).toContain('Página 2 de 2')
    expect(textos(r).join('\n')).not.toContain('"itens"')
  })

  it('cinemas obter usa o id no caminho', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { id: 'abc', nome: 'Cine' }))
    await despachar('cinemas obter abc', ctx(fn))
    expect(chamadas[0].url).toBe('/catalogo/api/v1/cinemas/abc')
  })

  it('salas listar filtra por cinema_id e sessoes listar por data', async () => {
    const a = fakeFetch(() => respostaJson(200, {}))
    await despachar('salas listar cinema_id=11111111-1111-1111-1111-111111111111', ctx(a.fn))
    expect(a.chamadas[0].url).toBe('/catalogo/api/v1/salas?cinema_id=11111111-1111-1111-1111-111111111111')
    const b = fakeFetch(() => respostaJson(200, {}))
    await despachar('sessoes listar data=2026-11-20', ctx(b.fn))
    expect(b.chamadas[0].url).toBe('/catalogo/api/v1/sessoes?data=2026-11-20')
  })

  it('health usa a raiz', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { status: 'ok', dependencias: { banco: 'ok' } }))
    const r = await despachar('health', ctx(fn))
    expect(chamadas[0].url).toBe('/catalogo/health')
    expect(r.linhas[0].tipo).toBe('tabela')
    expect(r.linhas[0].texto).toContain('| Serviço | ok    |')
    expect(r.linhas[0].texto).toContain('| banco   | ok    |')
  })

  it('controller e ação não diferenciam maiúsculas', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    await despachar('  FILMES   Listar ', ctx(fn))
    expect(chamadas[0].url).toBe('/catalogo/api/v1/filmes')
  })

  it('404 vira erro com título e detalhe do serviço', async () => {
    const { fn } = fakeFetch(() =>
      respostaJson(404, { type: 't', title: 'Recurso inexistente', status: 404, detail: 'sala não encontrada' }),
    )
    const r = await despachar('salas obter x', ctx(fn))
    expect(textos(r)).toEqual(['404 Recurso inexistente', '  sala não encontrada'])
    expect(r.linhas.every((l) => l.tipo === 'erro')).toBe(true)
  })

  it('rede indisponível vira mensagem e não lança', async () => {
    const r = await despachar('filmes listar', ctx(fetchQueFalha()))
    expect(textos(r)[0]).toContain('serviço indisponível')
  })

  it('argumentos inválidos não chamam o serviço e mostram o uso', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('filmes obter', ctx(fn))
    expect(chamadas).toHaveLength(0)
    expect(textos(r)).toEqual(['identificador ausente', 'uso: filmes obter <id>'])
  })
})

describe('despachar - comandos locais', () => {
  it('clear limpa a tela e desconhecidos dão erro, sem chamar o serviço', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    expect((await despachar('clear', ctx(fn))).limpar).toBe(true)
    expect(textos(await despachar('xyz', ctx(fn)))).toEqual(['comando não encontrado: xyz'])
    expect((await despachar('   ', ctx(fn))).linhas).toEqual([])
    expect(chamadas).toHaveLength(0)
  })

  it('echo, date, whoami e about foram removidos', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    for (const nome of ['echo oi', 'date', 'whoami', 'about']) {
      const r = await despachar(nome, ctx(fn))
      expect(textos(r)).toEqual([`comando não encontrado: ${nome.split(' ')[0]}`])
    }
    expect(chamadas).toHaveLength(0)
  })
})

describe('despachar - autenticação', () => {
  it('escrita sem token orienta o login e não chama o serviço', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, {}))
    const r = await despachar('filmes criar {"titulo":"X"}', ctx(fn))
    expect(chamadas).toHaveLength(0)
    expect(textos(r).join('\n')).toContain('use: login')
    expect(r.sessao).toBeUndefined()
  })

  it('com token envia Authorization', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    await despachar('filmes atualizar abc {"titulo":"Y"}', ctx(fn, comToken))
    expect(chamadas[0].init.headers).toHaveProperty('Authorization', 'Bearer tok')
  })

  it('401 devolve sessão sem token e orienta a autenticar de novo', async () => {
    const { fn } = fakeFetch(() => respostaJson(401, { title: 'Token expirado', status: 401 }))
    const r = await despachar('filmes remover abc', ctx(fn, comToken))
    expect(r.sessao).toEqual({ token: null })
    expect(textos(r).join('\n')).toContain('use login')
  })

  it('login válido devolve a sessão com token', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { access_token: 'novo' }))
    const r = await despachar('login meu-client meu-segredo', ctx(fn))
    expect(r.sessao).toEqual({ token: 'novo' })
    expect(chamadas[0].url).toContain('/identidade/realms/cinema/')
    expect(textos(r).join('\n')).not.toContain('meu-segredo')
  })

  it('login inválido não devolve sessão', async () => {
    const { fn } = fakeFetch(() => respostaJson(401, {}))
    const r = await despachar('login c errado', ctx(fn))
    expect(r.sessao).toBeUndefined()
  })

  it('login sem argumentos mostra o uso', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('login so-o-id', ctx(fn))
    expect(chamadas).toHaveLength(0)
    expect(textos(r)).toEqual(['uso: login | login <client_id> <client_secret>'])
  })

  it('login sem argumentos e com oidc pede o redirecionamento ao Keycloak', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const dados = new Map<string, string>()
    const oidc = {
      urlKeycloak: 'http://keycloak:8081',
      redirectUri: 'http://localhost:5173/',
      armazem: {
        getItem: (k: string) => dados.get(k) ?? null,
        setItem: (k: string, v: string) => void dados.set(k, v),
        removeItem: (k: string) => void dados.delete(k),
      },
    }
    const r = await despachar('login', { ...ctx(fn), oidc })
    expect(chamadas).toHaveLength(0)
    expect(r.redirecionar).toContain('http://keycloak:8081/realms/cinema/protocol/openid-connect/auth?')
    expect(r.sessao).toBeUndefined()
  })

  it('logout devolve sessão sem token', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('logout', ctx(fn, comToken))
    expect(r.sessao).toEqual({ token: null })
  })

  it('comandos que não mudam a sessão não trazem sessao', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    expect((await despachar('filmes listar', ctx(fn, comToken))).sessao).toBeUndefined()
    expect((await despachar('clear', ctx(fn, comToken))).sessao).toBeUndefined()
  })
})

describe('despachar - escritas', () => {
  it('filmes criar envia o corpo e mostra o Location', async () => {
    const { fn, chamadas } = fakeFetch(() =>
      respostaJson(201, { id: '9', titulo: 'X' }, { Location: '/api/v1/filmes/9' }),
    )
    const r = await despachar(
      'filmes criar {"titulo":"X","duracao_minutos":90,"classificacao_etaria":"Livre","genero":"Drama"}',
      ctx(fn, comToken),
    )
    expect(chamadas[0].init.method).toBe('POST')
    expect(JSON.parse(String(chamadas[0].init.body))).toMatchObject({ titulo: 'X', duracao_minutos: 90 })
    expect(textos(r)[0]).toBe('Cadastrado com sucesso.')
    expect(textos(r)).toContain('Localização: /api/v1/filmes/9')
  })

  it('salas atualizar usa PUT com id e corpo', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { id: 's1' }))
    await despachar('salas atualizar s1 {"numero":3,"tipo_tela":"IMAX"}', ctx(fn, comToken))
    expect(chamadas[0].url).toBe('/catalogo/api/v1/salas/s1')
    expect(chamadas[0].init.method).toBe('PUT')
  })

  it('sessoes remover confirma o 204', async () => {
    const { fn, chamadas } = fakeFetch(() => new Response(null, { status: 204 }))
    const r = await despachar('sessoes remover s1', ctx(fn, comToken))
    expect(chamadas[0].init.method).toBe('DELETE')
    expect(textos(r)).toEqual(['Removido com sucesso.'])
  })

  it('JSON inválido não chama o serviço', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, {}))
    const r = await despachar('filmes criar {titulo', ctx(fn, comToken))
    expect(chamadas).toHaveLength(0)
    expect(textos(r)[0].startsWith('JSON inválido')).toBe(true)
  })

  it('400 mostra os erros por campo', async () => {
    const { fn } = fakeFetch(() =>
      respostaJson(400, {
        title: 'Entrada inválida', status: 400,
        errors: [{ campo: 'titulo', mensagem: 'obrigatório' }, { campo: 'genero', mensagem: 'obrigatório' }],
      }),
    )
    const r = await despachar('filmes criar {"titulo":""}', ctx(fn, comToken))
    expect(textos(r)).toEqual(['400 Entrada inválida', '  titulo: obrigatório', '  genero: obrigatório'])
  })

  it('409 mostra o conflito do serviço', async () => {
    const { fn } = fakeFetch(() =>
      respostaJson(409, { title: 'Conflito', status: 409, detail: 'número de sala repetido' }),
    )
    const r = await despachar('salas criar {"numero":1}', ctx(fn, comToken))
    expect(textos(r)).toEqual(['409 Conflito', '  número de sala repetido'])
  })
})

describe('despachar - help e uso', () => {
  it('help junta os comandos locais e os de API', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    const t = textos(await despachar('help', ctx(fn))).join('\n')
    for (const removido of ['echo', 'whoami', 'about']) expect(t).not.toContain(removido)
    for (const c of ['help', 'clear', 'filmes listar', 'sessoes remover <id>', 'health', 'login', 'logout']) {
      expect(t).toContain(c)
    }
  })

  it('controller sem ação lista as ações válidas', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    expect(textos(await despachar('filmes', ctx(fn)))).toEqual([
      'ações de filmes: listar, criar, obter, atualizar, remover',
    ])
    expect(textos(await despachar('filmes xpto', ctx(fn)))).toEqual([
      'ação desconhecida: xpto',
      'ações de filmes: listar, criar, obter, atualizar, remover',
    ])
    expect(chamadas).toHaveLength(0)
  })
})
