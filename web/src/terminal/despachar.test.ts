import { describe, expect, it } from 'vitest'
import { BASE_TESTE, fakeFetch, fetchQueFalha, respostaJson } from './api/fakes'
import { SEM_SESSAO, type Sessao } from './api/sessao'
import { despachar, continuarDialogo, type Contexto } from './despachar'
import type { Resultado } from './comandos'
import type { Dialogo } from './dialogo'

function ctx(fetchFn: typeof fetch, sessao: Sessao = SEM_SESSAO): Contexto {
  return { fetchFn, base: BASE_TESTE, sessao }
}
const comToken: Sessao = { token: 'tok' }
const textos = (r: { linhas: { texto: string }[] }) => r.linhas.map((l) => l.texto)

/** Abre o comando e responde, uma a uma, às perguntas do diálogo. */
async function preencher(comando: string, respostas: string[], c: Contexto): Promise<Resultado> {
  let r = await despachar(comando, c)
  let d: Dialogo | null | undefined = r.dialogo
  for (const resposta of respostas) {
    if (!d) throw new Error('diálogo encerrado antes das respostas acabarem')
    r = await continuarDialogo(d, resposta, c)
    d = r.dialogo
  }
  return r
}

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
    const r = await despachar('filmes criar', ctx(fn))
    expect(chamadas).toHaveLength(0)
    expect(textos(r).join('\n')).toContain('use: login')
    expect(r.sessao).toBeUndefined()
    expect(r.dialogo).toBeUndefined()
  })

  it('com token envia Authorization', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    await despachar('filmes remover abc', ctx(fn, comToken))
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

const RESP_FILME = ['X', '', '90', 'Livre', 'Drama', '', '', 's']
const RESP_CINEMA = ['Cine', 'Floripa', 'sc', 'Rua A, 1', '', 's']

describe('despachar - escritas guiadas por campos', () => {
  it('filmes criar abre o diálogo sem chamar o serviço', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, {}))
    const r = await despachar('filmes criar', ctx(fn, comToken))
    expect(chamadas).toHaveLength(0)
    expect(r.dialogo).toBeTruthy()
    expect(textos(r).join('\n')).toContain('Título (obrigatório):')
  })

  it('filmes criar monta o corpo, envia e mostra o Location', async () => {
    const { fn, chamadas } = fakeFetch(() =>
      respostaJson(201, { id: '9', titulo: 'X' }, { Location: '/api/v1/filmes/9' }),
    )
    const r = await preencher('filmes criar', RESP_FILME, ctx(fn, comToken))
    expect(chamadas).toHaveLength(1)
    expect(chamadas[0].url).toBe('/catalogo/api/v1/filmes')
    expect(chamadas[0].init.method).toBe('POST')
    expect(JSON.parse(String(chamadas[0].init.body))).toEqual({
      titulo: 'X',
      duracao_minutos: 90,
      classificacao_etaria: 'Livre',
      genero: 'Drama',
    })
    expect(textos(r)).toContain('Cadastrado com sucesso.')
    expect(textos(r)).toContain('Localização: /api/v1/filmes/9')
    expect(r.dialogo).toBeNull()
  })

  it('cinemas criar envia estado em maiúsculas e omite ativo em branco', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, { id: 'c1' }))
    await preencher('cinemas criar', RESP_CINEMA, ctx(fn, comToken))
    expect(JSON.parse(String(chamadas[0].init.body))).toEqual({
      nome: 'Cine',
      cidade: 'Floripa',
      estado: 'SC',
      endereco: 'Rua A, 1',
    })
  })

  it('sessoes criar envia preco_base como string', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, { id: 's1' }))
    const f = '11111111-1111-1111-1111-111111111111'
    const s = '22222222-2222-2222-2222-222222222222'
    await preencher('sessoes criar', [f, s, '2026-12-01 19:30', 'dublado', '42,5', '', 's'], ctx(fn, comToken))
    const corpo = JSON.parse(String(chamadas[0].init.body)) as Record<string, unknown>
    expect(corpo).toEqual({
      filme_id: f,
      sala_id: s,
      data_hora_inicio: new Date(2026, 11, 1, 19, 30).toISOString(),
      idioma: 'DUBLADO',
      preco_base: '42.5',
    })
  })

  it('salas criar envia as fileiras no formato do contrato', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, { id: 'sa1' }))
    const cinema = '11111111-1111-1111-1111-111111111111'
    await preencher(
      'salas criar',
      [cinema, '3', 'imax', 'a', '10', '', 's', 'B', '12', 'pcd', 'n', '', 's'],
      ctx(fn, comToken),
    )
    expect(JSON.parse(String(chamadas[0].init.body))).toEqual({
      cinema_id: cinema,
      numero: 3,
      tipo_tela: 'IMAX',
      fileiras: [
        { fileira: 'A', assentos: 10, tipo: 'NORMAL' },
        { fileira: 'B', assentos: 12, tipo: 'PCD' },
      ],
    })
  })

  it('salas atualizar pede só número, tela e ativo e usa PUT com id', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, { id: 's1' }))
    const inicio = await despachar('salas atualizar s1', ctx(fn, comToken))
    const perguntas = textos(inicio).join('\n')
    expect(perguntas).toContain('Número da sala')
    expect(perguntas).not.toContain('cinema')
    await preencher('salas atualizar s1', ['3', 'imax', '', 's'], ctx(fn, comToken))
    expect(chamadas[0].url).toBe('/catalogo/api/v1/salas/s1')
    expect(chamadas[0].init.method).toBe('PUT')
    expect(JSON.parse(String(chamadas[0].init.body))).toEqual({ numero: 3, tipo_tela: 'IMAX' })
  })

  it('atualizar sem id mostra o uso; com JSON na linha recusa sem abrir diálogo', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const a = await despachar('filmes atualizar', ctx(fn, comToken))
    expect(textos(a)).toEqual(['identificador ausente', 'uso: filmes atualizar <id>'])
    const b = await despachar('filmes criar {"titulo":"X"}', ctx(fn, comToken))
    expect(textos(b)[0]).toContain('não recebe JSON')
    expect(b.dialogo).toBeUndefined()
    const c = await despachar('filmes atualizar abc {"titulo":"Y"}', ctx(fn, comToken))
    expect(textos(c)[0]).toContain('não recebe JSON')
    expect(chamadas).toHaveLength(0)
  })

  it('cancelar encerra sem nenhuma requisição', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, {}))
    const r = await preencher('filmes criar', ['X', 'cancelar'], ctx(fn, comToken))
    expect(chamadas).toHaveLength(0)
    expect(r.dialogo).toBeNull()
    expect(textos(r).join('\n')).toContain('Nada foi enviado')
  })

  it('recusar a confirmação não envia', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(201, {}))
    const r = await preencher('filmes criar', [...RESP_FILME.slice(0, -1), 'n'], ctx(fn, comToken))
    expect(chamadas).toHaveLength(0)
    expect(r.dialogo).toBeNull()
  })

  it('400 do serviço mostra os erros por campo e encerra o diálogo', async () => {
    const { fn } = fakeFetch(() =>
      respostaJson(400, {
        title: 'Entrada inválida', status: 400,
        errors: [{ campo: 'titulo', mensagem: 'obrigatório' }, { campo: 'genero', mensagem: 'obrigatório' }],
      }),
    )
    const r = await preencher('filmes criar', RESP_FILME, ctx(fn, comToken))
    expect(textos(r).slice(-3)).toEqual(['400 Entrada inválida', '  titulo: obrigatório', '  genero: obrigatório'])
    expect(r.dialogo).toBeNull()
  })

  it('409 mostra o conflito do serviço', async () => {
    const { fn } = fakeFetch(() =>
      respostaJson(409, { title: 'Conflito', status: 409, detail: 'número de sala repetido' }),
    )
    const cinema = '11111111-1111-1111-1111-111111111111'
    const r = await preencher('salas criar', [cinema, '1', '2d', 'A', '5', '', 'n', '', 's'], ctx(fn, comToken))
    expect(textos(r).slice(-2)).toEqual(['409 Conflito', '  número de sala repetido'])
  })

  it('401 no envio zera a sessão', async () => {
    const { fn } = fakeFetch(() => respostaJson(401, { title: 'Token expirado', status: 401 }))
    const r = await preencher('filmes criar', RESP_FILME, ctx(fn, comToken))
    expect(r.sessao).toEqual({ token: null })
  })
})

describe('despachar - remover', () => {
  it('sessoes remover confirma o 204', async () => {
    const { fn, chamadas } = fakeFetch(() => new Response(null, { status: 204 }))
    const r = await despachar('sessoes remover s1', ctx(fn, comToken))
    expect(chamadas[0].init.method).toBe('DELETE')
    expect(textos(r)).toEqual(['Removido com sucesso.'])
  })
})

describe('despachar - help e uso', () => {
  it('help mostra só os nomes dos controllers, em até 20 linhas', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('help', ctx(fn))
    const t = textos(r).join('\n')
    for (const removido of ['echo', 'whoami', 'about']) expect(t).not.toContain(removido)
    for (const c of ['help', 'clear', 'filmes', 'cinemas', 'salas', 'sessoes', 'health', 'login', 'logout']) {
      expect(t).toContain(c)
    }
    expect(t).not.toContain('sessoes remover <id>')
    expect(t).not.toContain('filmes listar')
    expect(r.linhas.length).toBeLessThanOrEqual(20)
    expect(chamadas).toHaveLength(0)
  })

  it('help <controller> mostra o mesmo help geral, sem erro', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    const geral = textos(await despachar('help', ctx(fn)))
    expect(textos(await despachar('help filmes', ctx(fn)))).toEqual(geral)
  })

  it.each(['filmes', 'cinemas', 'salas', 'sessoes'])('%s help mostra só as ações do controller, sem requisição', async (c) => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const t = textos(await despachar(`${c} help`, ctx(fn))).join('\n')
    for (const acao of ['listar', 'criar', 'obter', 'atualizar', 'remover']) expect(t).toContain(`${c} ${acao}`)
    for (const outro of ['filmes', 'cinemas', 'salas', 'sessoes'].filter((x) => x !== c)) {
      expect(t).not.toContain(`${outro} listar`)
    }
    expect(chamadas).toHaveLength(0)
  })

  it('health help mostra só o health, sem requisição', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const t = textos(await despachar('health help', ctx(fn))).join('\n')
    expect(t).toContain('health')
    expect(t).toContain('apto a receber tráfego')
    expect(t).not.toContain('filmes')
    expect(chamadas).toHaveLength(0)
  })

  it('ignora caixa, espaços e texto extra em <controller> help', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const esperado = textos(await despachar('filmes help', ctx(fn)))
    expect(textos(await despachar('FILMES   help', ctx(fn)))).toEqual(esperado)
    expect(textos(await despachar('filmes HELP', ctx(fn)))).toEqual(esperado)
    expect(textos(await despachar('filmes help extra', ctx(fn)))).toEqual(esperado)
    expect(chamadas).toHaveLength(0)
  })

  it('clear help mantém o comportamento atual: clear ignora o resto', async () => {
    const { fn } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('clear help', ctx(fn))
    expect(r.limpar).toBe(true)
  })

  it('login help mantém o comportamento atual (uso do login)', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    const r = await despachar('login help', ctx(fn))
    expect(textos(r)).toEqual(['uso: login | login <client_id> <client_secret>'])
    expect(chamadas).toHaveLength(0)
  })

  it('controller sem ação lista as ações válidas e a dica do help', async () => {
    const { fn, chamadas } = fakeFetch(() => respostaJson(200, {}))
    expect(textos(await despachar('filmes', ctx(fn)))).toEqual([
      'ações de filmes: listar, criar, obter, atualizar, remover',
      'use: filmes help para ver os detalhes.',
    ])
    expect(textos(await despachar('filmes xpto', ctx(fn)))).toEqual([
      'ação desconhecida: xpto',
      'ações de filmes: listar, criar, obter, atualizar, remover',
      'use: filmes help para ver os detalhes.',
    ])
    expect(chamadas).toHaveLength(0)
  })
})
