import { describe, expect, it } from 'vitest'
import { executar } from './comandos'

const agora = new Date(2026, 10, 14, 9, 30, 0)
const textos = (entrada: string) => executar(entrada, agora).linhas.map((l) => l.texto)

describe('executar', () => {
  it('help lista os comandos disponíveis', () => {
    const t = textos('help').join('\n')
    for (const nome of ['help', 'clear', 'echo', 'date', 'whoami', 'about']) {
      expect(t).toContain(nome)
    }
  })

  it('echo repete o texto, inclusive marcação, literalmente', () => {
    expect(textos('echo <b>oi</b>')).toEqual(['<b>oi</b>'])
  })

  it('echo sem argumento produz uma linha vazia', () => {
    expect(textos('echo')).toEqual([''])
  })

  it('date usa a data informada em pt-BR', () => {
    const t = textos('date')
    expect(t).toHaveLength(1)
    expect(t[0]).toBe(agora.toLocaleString('pt-BR'))
  })

  it('whoami responde visitante', () => {
    expect(textos('whoami')).toEqual(['visitante'])
  })

  it('about descreve o terminal', () => {
    expect(textos('about').length).toBeGreaterThan(0)
  })

  it('comando desconhecido gera erro com o nome digitado', () => {
    const { linhas } = executar('xyz argumento', agora)
    expect(linhas).toEqual([{ tipo: 'erro', texto: 'comando não encontrado: xyz' }])
  })

  it('entrada vazia ou só de espaços não produz saída', () => {
    expect(executar('', agora)).toEqual({ linhas: [], limpar: false })
    expect(executar('   ', agora)).toEqual({ linhas: [], limpar: false })
  })

  it('ignora maiúsculas e espaços extras', () => {
    expect(textos('  WHOAMI  ')).toEqual(['visitante'])
    expect(textos('  Echo    ola  mundo ')).toEqual(['ola  mundo'])
  })

  it('clear devolve limpar: true e nenhuma linha', () => {
    expect(executar('clear', agora)).toEqual({ linhas: [], limpar: true })
  })
})
