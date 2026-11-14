import { describe, expect, it } from 'vitest'
import { FalhaRequisicao } from './cliente'
import {
  desenharTabela,
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

const tabela = (...linhas: string[]) => linhas.join('\n')

describe('formatarResposta - tabelas', () => {
  it('a ajuda não traz exemplos com JSON e explica o fluxo guiado', () => {
    const t = textos(formatarAjuda()).join('\n')
    expect(t).not.toContain('<json>')
    expect(t).toContain('perguntam os campos')
    expect(t).toContain('filmes criar')
  })

  it('um filme vira uma tabela Campo/Valor, com sinopse quebrada e sem JSON', () => {
    const r = formatarResposta(def('filmes', 'obter'), {
      status: 200,
      corpo: { ...filme, sinopse: 'Um hacker descobre a verdade sobre a realidade e se une a rebeldes para lutar contra as máquinas.' },
    })
    expect(r).toHaveLength(1)
    expect(r[0].tipo).toBe('tabela')
    expect(r[0].texto).toBe(
      tabela(
        '+---------------+-----------------------------------------------------------+',
        '| Campo         | Valor                                                     |',
        '+---------------+-----------------------------------------------------------+',
        '| Id            | f1                                                        |',
        '| Título        | The Matrix                                                |',
        '| Duração       | 130 min                                                   |',
        '| Classificação | 14                                                        |',
        '| Gênero        | Ficção                                                    |',
        '| Status        | Em cartaz                                                 |',
        '| Sinopse       | Um hacker descobre a verdade sobre a realidade e se une a |',
        '|               | rebeldes para lutar contra as máquinas.                   |',
        '| Imagem        | https://example.com/m.png                                 |',
        '+---------------+-----------------------------------------------------------+',
      ),
    )
    expect(r[0].texto).not.toMatch(/[{}"]/)
  })

  it('lista de filmes: uma linha por registro, título cortado, resumo da paginação', () => {
    const r = formatarResposta(def('filmes', 'listar'), {
      status: 200,
      corpo: {
        itens: [filme, { ...filme, id: 'f2', titulo: 'Um Sonho de Liberdade Muito Longo Para Caber Aqui', status: 'BREVE' }],
        pagina: { pagina: 1, tamanho: 2, total: 5, tem_proxima: true },
      },
    })
    expect(r[0].texto).toBe(
      tabela(
        '+----+----------------------------------+---------+---------------+--------+-----------+',
        '| Id | Título                           | Duração | Classificação | Gênero | Status    |',
        '+----+----------------------------------+---------+---------------+--------+-----------+',
        '| f1 | The Matrix                       | 130 min | 14            | Ficção | Em cartaz |',
        '| f2 | Um Sonho de Liberdade Muito Lon… | 130 min | 14            | Ficção | Em breve  |',
        '+----+----------------------------------+---------+---------------+--------+-----------+',
      ),
    )
    expect(textos(r).slice(1)).toEqual(['Página 1 de 3 · 5 registros no total', 'Há mais resultados: use page=2.'])
  })

  it('lista vazia não desenha tabela', () => {
    const r = formatarResposta(def('cinemas', 'listar'), {
      status: 200,
      corpo: { itens: [], pagina: { pagina: 1, tamanho: 20, total: 0, tem_proxima: false } },
    })
    expect(r.some((l) => l.tipo === 'tabela')).toBe(false)
    expect(textos(r)).toEqual(['Nenhum registro encontrado.', 'Página 1 de 1 · 0 registros no total'])
  })

  it('lista de cinemas', () => {
    const r = formatarResposta(def('cinemas', 'listar'), {
      status: 200,
      corpo: {
        itens: [{ id: 'c1', nome: 'Cine Centro', cidade: 'Florianópolis', estado: 'SC', endereco: 'Rua A, 10', ativo: true }],
        pagina: { pagina: 1, tamanho: 20, total: 1, tem_proxima: false },
      },
    })
    expect(r[0].texto).toBe(
      tabela(
        '+----+-------------+-----------+------------------+----------+',
        '| Id | Nome        | Endereço  | Cidade/UF        | Situação |',
        '+----+-------------+-----------+------------------+----------+',
        '| c1 | Cine Centro | Rua A, 10 | Florianópolis/SC | Ativo    |',
        '+----+-------------+-----------+------------------+----------+',
      ),
    )
  })

  it('lista de salas com planta e capacidade', () => {
    const r = formatarResposta(def('salas', 'listar'), {
      status: 200,
      corpo: {
        itens: [{
          id: 's1', cinema_id: 'c1', numero: 3, tipo_tela: 'IMAX', capacidade_total: 22, ativo: false,
          fileiras: [{ fileira: 'A', assentos: 10 }, { fileira: 'B', assentos: 2, tipo: 'PCD' }],
        }],
        pagina: { pagina: 1, tamanho: 20, total: 1, tem_proxima: false },
      },
    })
    expect(r[0].texto).toBe(
      tabela(
        '+----+--------+------+---------+----------+-------------------+--------+',
        '| Id | Número | Tela | Lugares | Situação | Fileiras          | Cinema |',
        '+----+--------+------+---------+----------+-------------------+--------+',
        '| s1 | 3      | IMAX | 22      | Inativa  | A (10), B (2 PCD) | c1     |',
        '+----+--------+------+---------+----------+-------------------+--------+',
      ),
    )
  })

  it('grade de sessões com data, idioma e preço em pt-BR', () => {
    const r = formatarResposta(
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
    )
    const linhas = r[0].texto.split('\n')
    expect(linhas[1]).toBe('| Id | Filme      | Cinema      | Sala   | Início            | Idioma    | Preço    |')
    expect(linhas[3]).toMatch(/^\| x1 \| The Matrix \| Cine Centro \| 3 \(3D\) \| 20\/11\/2026,? 19:30 \| Legendado \| R\$ 42,00 \|$/)
    expect(linhas[3]).toContain('20/11/2026')
    expect(linhas[3]).toContain('19:30')
  })

  it('uma sessão da grade (obter) inclui os ids do filme e do cinema', () => {
    const r = formatarResposta(def('sessoes', 'obter'), {
      status: 200,
      corpo: {
        id: 'x1', filme_id: 'f1', filme_titulo: 'The Matrix', cinema_id: 'c1', cinema_nome: 'Cine Centro',
        sala_numero: 3, tipo_tela: '3D', data_hora_inicio: '2026-11-20T19:30:00Z', idioma: 'LEGENDADO', preco_base: '42.00',
      },
    })
    expect(r[0].texto).toContain('| Filme (id)  | f1')
    expect(r[0].texto).toContain('| Cinema (id) | c1')
  })

  it('sessão gravada em tabela Campo/Valor', () => {
    const r = formatarResposta(
      def('sessoes', 'obter'),
      {
        status: 200,
        corpo: {
          id: 'x1', filme_id: 'f1', sala_id: 's1', data_hora_inicio: '2026-11-20T19:30:00Z',
          idioma: 'DUBLADO', preco_base: '30.5', status: 'AGENDADA',
        },
      },
      { fuso: 'UTC' },
    )
    const t = r[0].texto
    expect(t).toContain('| Idioma | Dublado')
    expect(t).toContain('| Preço  | R$ 30,5')
    expect(t).toContain('| Status | Agendada')
    expect(t).toContain('| Filme  | f1')
    expect(t).toContain('| Sala   | s1')
  })

  it('health em tabela', () => {
    const r = formatarResposta(def('operacional', 'health'), {
      status: 200,
      corpo: { status: 'ok', dependencias: { banco: 'ok' } },
    })
    expect(r[0].texto).toBe(
      tabela(
        '+---------+-------+',
        '| Campo   | Valor |',
        '+---------+-------+',
        '| Serviço | ok    |',
        '| banco   | ok    |',
        '+---------+-------+',
      ),
    )
  })

  it('criar e atualizar anunciam o sucesso e mostram a tabela; criar mostra a localização', () => {
    const c = formatarResposta(def('filmes', 'criar'), { status: 201, corpo: filme, location: '/api/v1/filmes/f1' })
    expect(c[0]).toEqual({ tipo: 'sistema', texto: 'Cadastrado com sucesso.' })
    expect(c[1].tipo).toBe('tabela')
    expect(c[c.length - 1]).toEqual({ tipo: 'sistema', texto: 'Localização: /api/v1/filmes/f1' })
    const u = formatarResposta(def('filmes', 'atualizar'), { status: 200, corpo: filme })
    expect(u[0]).toEqual({ tipo: 'sistema', texto: 'Atualizado com sucesso.' })
  })

  it('204 confirma a remoção', () => {
    expect(textos(formatarResposta(def('filmes', 'remover'), { status: 204, corpo: null }))).toEqual([
      'Removido com sucesso.',
    ])
  })

  it('corpo de texto cai em linhas simples', () => {
    expect(
      textos(formatarResposta(def('operacional', 'health'), { status: 200, corpo: 'texto qualquer' })),
    ).toEqual(['texto qualquer'])
  })

  it('mantém marcação como texto literal', () => {
    const r = formatarResposta(def('filmes', 'obter'), { status: 200, corpo: { ...filme, titulo: '<b>oi</b>' } })
    expect(r[0].texto).toContain('| Título        | <b>oi</b>')
  })
})

describe('desenharTabela', () => {
  it('alinha colunas e células com várias linhas', () => {
    expect(desenharTabela(['A', 'BB'], [['x', 'um\ndois'], ['yyy', '']])).toBe(
      tabela(
        '+-----+------+',
        '| A   | BB   |',
        '+-----+------+',
        '| x   | um   |',
        '|     | dois |',
        '| yyy |      |',
        '+-----+------+',
      ),
    )
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
