import { describe, expect, it } from 'vitest'
import { FORMULARIOS } from './api/formularios'
import { buscarDefinicao } from './api/tabela'
import { cancelar, iniciar, registrarNoHistorico, responder, type Dialogo, type Passo } from './dialogo'

function abrir(controller: string, acao: string, id?: string): Passo {
  const def = buscarDefinicao(controller, acao)
  if (!def) throw new Error('definição inexistente')
  return iniciar(def, FORMULARIOS[`${controller}.${acao}`], id)
}
const textos = (p: Passo) => p.linhas.map((l) => l.texto)

function seguir(p: Passo, respostas: string[]): Passo {
  let atual = p
  for (const r of respostas) {
    if (!atual.dialogo) throw new Error('encerrado')
    atual = responder(atual.dialogo, r)
  }
  return atual
}

describe('diálogo - campos', () => {
  it('a primeira pergunta mostra nome, obrigatoriedade e a dica de cancelar', () => {
    const t = textos(abrir('filmes', 'criar')).join('\n')
    expect(t).toContain('cancelar')
    expect(t).toContain('Título (obrigatório):')
  })

  it('pergunta mostra opções e padrão dos campos de opção e lógicos', () => {
    const p = seguir(abrir('filmes', 'criar'), ['X', '', '90', 'Livre', 'Drama', ''])
    expect(textos(p)).toEqual(['Status (opcional, padrão EM_CARTAZ) [EM_CARTAZ|BREVE|FORA_DE_CARTAZ]:'])
    const c = seguir(abrir('cinemas', 'criar'), ['n', 'c', 'sc', 'r'])
    expect(textos(c)).toEqual(['Ativo (opcional, padrão true) [s/n]:'])
  })

  it('resposta inválida repete a mesma pergunta sem avançar', () => {
    const p = seguir(abrir('filmes', 'criar'), ['X', '', 'abc'])
    expect(textos(p)).toEqual(['informe um número inteiro', 'Duração em minutos (obrigatório):'])
    expect(p.dialogo?.indice).toBe(2)
  })

  it('campo obrigatório em branco é recusado', () => {
    const p = seguir(abrir('filmes', 'criar'), [''])
    expect(textos(p)[0]).toBe('campo obrigatório')
    expect(p.dialogo?.indice).toBe(0)
  })

  it('opcional em branco avança; ao fim mostra o resumo e pede confirmação', () => {
    const p = seguir(abrir('filmes', 'criar'), ['X', '', '90', 'Livre', 'Drama', '', ''])
    const t = textos(p)
    expect(t).toContain('Resumo:')
    expect(t).toContain('  titulo: X')
    expect(t).not.toContain('  sinopse: ')
    expect(t[t.length - 1]).toBe('Enviar? [s/n]:')
    expect(p.dialogo?.etapa).toBe('confirmacao')
  })

  it('confirmar com s devolve o corpo; resposta estranha repete; n não envia', () => {
    const resumo = seguir(abrir('cinemas', 'criar'), ['Cine', 'Floripa', 'sc', 'Rua A', ''])
    const estranho = responder(resumo.dialogo as Dialogo, 'talvez')
    expect(estranho.enviar).toBeUndefined()
    expect(estranho.dialogo?.etapa).toBe('confirmacao')
    const sim = responder(resumo.dialogo as Dialogo, 's')
    expect(sim.enviar?.corpo).toEqual({ nome: 'Cine', cidade: 'Floripa', estado: 'SC', endereco: 'Rua A' })
    expect(sim.dialogo).toBeNull()
    const nao = responder(resumo.dialogo as Dialogo, 'n')
    expect(nao.enviar).toBeUndefined()
    expect(nao.dialogo).toBeNull()
  })
})

const CINEMA = '11111111-1111-1111-1111-111111111111'

describe('diálogo - fileiras', () => {
  const ate = ['cinema_id', '1', '2d']
  const sala = () => seguir(abrir('salas', 'criar'), [CINEMA, '1', '2d'])

  it('depois do tipo de tela pede a primeira fileira', () => {
    expect(textos(sala()).join('\n')).toContain('Fileiras da sala')
    expect(textos(sala()).join('\n')).toContain('Letra da fileira')
    expect(ate).toHaveLength(3)
  })

  it('percorre letra → assentos → tipo → mais; s repete, n avança', () => {
    let p = seguir(sala(), ['a', '10', ''])
    expect(textos(p)).toEqual(['Fileira A registrada.', 'Adicionar outra fileira? [s/n]:'])
    p = seguir(p, ['s'])
    expect(p.dialogo?.etapa).toBe('fileira-letra')
    p = seguir(p, ['B', '12', 'pcd', 'n'])
    expect(p.dialogo?.fileiras).toEqual([
      { fileira: 'A', assentos: 10, tipo: 'NORMAL' },
      { fileira: 'B', assentos: 12, tipo: 'PCD' },
    ])
    expect(textos(p)).toEqual(['Ativo (opcional, padrão true) [s/n]:'])
  })

  it('não permite pular a primeira fileira nem usar letras/assentos inválidos', () => {
    expect(textos(seguir(sala(), ['']))[0]).toBe('campo obrigatório')
    for (const ruim of ['A1', 'ABCDEFG', '1']) {
      expect(textos(seguir(sala(), [ruim]))[0]).toBe('use de 1 a 5 letras, de A a Z')
    }
    expect(textos(seguir(sala(), ['A', '0']))[0]).toBe('o mínimo é 1')
    expect(textos(seguir(sala(), ['A', '5', 'x']))[0]).toContain('opção inválida')
  })

  it('o corpo final leva as fileiras no formato do contrato', () => {
    const p = seguir(sala(), ['A', '5', '', 'n', '', 's'])
    expect(p.enviar?.corpo).toEqual({
      cinema_id: CINEMA,
      numero: 1,
      tipo_tela: '2D',
      fileiras: [{ fileira: 'A', assentos: 5, tipo: 'NORMAL' }],
    })
  })
})

describe('diálogo - cancelar e histórico', () => {
  it('cancelar em qualquer etapa encerra sem enviar', () => {
    const etapas: Passo[] = [
      abrir('filmes', 'criar'),
      seguir(abrir('salas', 'criar'), [CINEMA, '1', '2d', 'A']),
      seguir(abrir('salas', 'criar'), [CINEMA, '1', '2d', 'A', '5', '', 'n', '', '']),
    ]
    for (const e of etapas) {
      const r = responder(e.dialogo as Dialogo, ' CANCELAR ')
      expect(r.dialogo).toBeNull()
      expect(r.enviar).toBeUndefined()
      expect(textos(r)).toEqual(['Preenchimento cancelado. Nada foi enviado.'])
    }
    expect(cancelar().dialogo).toBeNull()
  })

  it('respostas do diálogo não vão ao histórico; comandos comuns vão', () => {
    expect(registrarNoHistorico(abrir('filmes', 'criar').dialogo)).toBe(false)
    expect(registrarNoHistorico(null)).toBe(true)
  })

  it('atualizar cita o id na introdução', () => {
    expect(textos(abrir('filmes', 'atualizar', 'abc'))[0]).toContain('filmes atualizar abc')
  })
})
