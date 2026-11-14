import { describe, expect, it } from 'vitest'
import { FalhaRequisicao } from './cliente'
import {
  formatarAcoes,
  formatarAjuda,
  formatarErro,
  formatarFalha,
  formatarResposta,
  formatarUso,
} from './formatar'
import { TABELA, buscarDefinicao } from './tabela'

const textos = (linhas: { texto: string }[]) => linhas.map((l) => l.texto)

const def = (c: string, a: string) => {
  const d = buscarDefinicao(c, a)
  if (!d) throw new Error('definição inexistente')
  return d
}

const filme = {
  id: 'f1', titulo: 'The Matrix', sinopse: 'Um hacker descobre a verdade.', duracao_minutos: 130,
  classificacao_etaria: '14', genero: 'Ficção', imagem_url: 'https://example.com/m.png', status: 'EM_CARTAZ',
}

describe('formatarResposta - amigável', () => {
  it('filme em texto legível, sem JSON', () => {
    const t = textos(formatarResposta(def('filmes', 'obter'), { status: 200, corpo: filme }))
    expect(t).toEqual([
      'The Matrix',
      '  130 min · 14 · Ficção · Em cartaz',
      '  Um hacker descobre a verdade.',
      '  Imagem: https://example.com/m.png',
      '  Id: f1',
    ])
    expect(t.join('\n')).not.toMatch(/[{}"]/)
  })

  it('lista com itens separados e resumo da paginação', () => {
    const t = textos(
      formatarResposta(def('filmes', 'listar'), {
        status: 200,
        corpo: {
          itens: [filme, { ...filme, id: 'f2', titulo: 'Outro', sinopse: undefined, imagem_url: undefined, status: 'BREVE' }],
          pagina: { pagina: 1, tamanho: 2, total: 5, tem_proxima: true },
        },
      }),
    )
    expect(t).toContain('')
    expect(t).toContain('Outro')
    expect(t).toContain('  130 min · 14 · Ficção · Em breve')
    expect(t.slice(-2)).toEqual(['Página 1 de 3 · 5 registros no total', 'Há mais resultados: use page=2.'])
  })

  it('lista vazia', () => {
    const t = textos(
      formatarResposta(def('cinemas', 'listar'), {
        status: 200,
        corpo: { itens: [], pagina: { pagina: 1, tamanho: 20, total: 0, tem_proxima: false } },
      }),
    )
    expect(t).toEqual(['Nenhum registro encontrado.', '', 'Página 1 de 1 · 0 registros no total'])
  })

  it('cinema', () => {
    const t = textos(
      formatarResposta(def('cinemas', 'obter'), {
        status: 200,
        corpo: { id: 'c1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A, 10', ativo: true },
      }),
    )
    expect(t).toEqual(['Cine Centro', '  Rua A, 10, Florianópolis/SC · Ativo', '  Id: c1'])
  })

  it('sala com planta e capacidade', () => {
    const t = textos(
      formatarResposta(def('salas', 'obter'), {
        status: 200,
        corpo: {
          id: 's1', cinema_id: 'c1', numero: 3, tipo_tela: 'IMAX', capacidade_total: 22, ativo: false,
          fileiras: [{ fileira: 'A', assentos: 10 }, { fileira: 'B', assentos: 2, tipo: 'PCD' }],
        },
      }),
    )
    expect(t).toEqual([
      'Sala 3 · IMAX · 22 lugares · Inativa',
      '  Fileiras: A (10), B (2 PCD)',
      '  Cinema: c1',
      '  Id: s1',
    ])
  })

  it('sessão na grade com data, preço e idioma', () => {
    const t = textos(
      formatarResposta(
        def('sessoes', 'listar'),
        {
          status: 200,
          corpo: {
            itens: [{
              id: 'x1', filme_id: 'f1', filme_titulo: 'The Matrix', cinema_id: 'c1', cinema_nome: 'Cine Centro',
              sala_numero: 3, tipo_tela: '3D', data_hora_inicio: '2026-11-20T19:30:00Z', idioma: 'LEGENDADO',
              preco_base: '42.00',
            }],
            pagina: { pagina: 1, tamanho: 20, total: 1, tem_proxima: false },
          },
        },
        { fuso: 'UTC' },
      ),
    )
    expect(t[0]).toMatch(/^The Matrix — 20\/11\/2026,? 19:30$/)
    expect(t[1]).toBe('  Cine Centro · Sala 3 (3D) · Legendado · R$ 42,00')
    expect(t).toContain('  Id: x1')
    expect(t).toContain('  Filme: f1 · Cinema: c1')
  })

  it('sessão gravada', () => {
    const t = textos(
      formatarResposta(
        def('sessoes', 'obter'),
        {
          status: 200,
          corpo: {
            id: 'x1', filme_id: 'f1', sala_id: 's1', data_hora_inicio: '2026-11-20T19:30:00Z',
            idioma: 'DUBLADO', preco_base: '30.5', status: 'AGENDADA',
          },
        },
        { fuso: 'UTC' },
      ),
    )
    expect(t[0]).toMatch(/^Sessão em 20\/11\/2026,? 19:30$/)
    expect(t[1]).toBe('  Dublado · R$ 30,5 · Agendada')
    expect(t.slice(2)).toEqual(['  Filme: f1', '  Sala: s1', '  Id: x1'])
  })

  it('health em texto', () => {
    const t = textos(
      formatarResposta(def('operacional', 'health'), { status: 200, corpo: { status: 'ok', dependencias: { banco: 'ok' } } }),
    )
    expect(t).toEqual(['Serviço: ok', '  banco: ok'])
  })

  it('criar e atualizar anunciam o sucesso; criar mostra a localização', () => {
    const c = textos(
      formatarResposta(def('filmes', 'criar'), { status: 201, corpo: filme, location: '/api/v1/filmes/f1' }),
    )
    expect(c[0]).toBe('Cadastrado com sucesso.')
    expect(c[c.length - 1]).toBe('Localização: /api/v1/filmes/f1')
    expect(textos(formatarResposta(def('filmes', 'atualizar'), { status: 200, corpo: filme }))[0]).toBe(
      'Atualizado com sucesso.',
    )
  })

  it('204 confirma a remoção', () => {
    expect(textos(formatarResposta(def('filmes', 'remover'), { status: 204, corpo: null }))).toEqual([
      'Removido com sucesso.',
    ])
  })

  it('formato desconhecido cai em chave: valor, sem chaves de JSON', () => {
    const t = textos(
      formatarResposta(def('filmes', 'obter'), { status: 200, corpo: undefined }),
    )
    expect(t).toEqual([])
    const g = textos(
      formatarResposta(def('operacional', 'health'), { status: 200, corpo: 'texto qualquer' }),
    )
    expect(g).toEqual(['texto qualquer'])
  })

  it('mantém marcação como texto literal', () => {
    const t = textos(
      formatarResposta(def('filmes', 'obter'), { status: 200, corpo: { ...filme, titulo: '<b>oi</b>' } }),
    )
    expect(t[0]).toBe('<b>oi</b>')
  })
})

describe('formatarErro', () => {
  it('mostra status, título, detalhe e erros por campo', () => {
    const linhas = formatarErro({
      status: 400,
      corpo: {
        type: 'x', title: 'Entrada inválida', status: 400, detail: 'corpo inválido',
        errors: [{ campo: 'titulo', mensagem: 'obrigatório' }],
      },
    })
    expect(textos(linhas)).toEqual(['400 Entrada inválida', '  corpo inválido', '  titulo: obrigatório'])
    expect(linhas.every((l) => l.tipo === 'erro')).toBe(true)
  })

  it('usa títulos padrão quando o serviço não envia problema', () => {
    expect(textos(formatarErro({ status: 404, corpo: null }))).toEqual(['404 Não encontrado'])
    expect(textos(formatarErro({ status: 409, corpo: null }))).toEqual(['409 Conflito'])
    expect(textos(formatarErro({ status: 401, corpo: null }))).toEqual(['401 Não autenticado'])
    expect(textos(formatarErro({ status: 503, corpo: { status: 'x' } }))[0]).toBe('503 Serviço indisponível')
  })

  it('corpo de texto aparece como detalhe', () => {
    expect(textos(formatarErro({ status: 405, corpo: 'Method Not Allowed' }))).toEqual([
      '405 Erro',
      '  Method Not Allowed',
    ])
  })
})

describe('formatarFalha', () => {
  it('distingue rede de timeout', () => {
    expect(textos(formatarFalha(new FalhaRequisicao('rede')))[0]).toContain('serviço indisponível')
    expect(textos(formatarFalha(new FalhaRequisicao('timeout')))[0]).toContain('tempo esgotado')
  })
})

describe('ajuda', () => {
  it('formatarAjuda lista os 21 comandos, login e logout', () => {
    const t = textos(formatarAjuda()).join('\n')
    for (const d of TABELA) expect(t).toContain(d.uso)
    expect(t).toContain('login <client_id> <client_secret>')
    expect(t).toContain('logout')
    for (const grupo of ['Filmes', 'Cinemas', 'Salas', 'Sessões', 'Operacional']) expect(t).toContain(grupo)
  })

  it('formatarUso mostra a sintaxe', () => {
    const def = buscarDefinicao('filmes', 'obter')
    if (!def) throw new Error('sem definição')
    expect(textos(formatarUso(def))).toEqual(['uso: filmes obter <id>'])
  })

  it('formatarAcoes lista as ações válidas', () => {
    expect(textos(formatarAcoes('filmes'))).toEqual(['ações de filmes: listar, criar, obter, atualizar, remover'])
  })
})
