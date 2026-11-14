import { describe, expect, it } from 'vitest'
import { executar } from './comandos'

const textos = (entrada: string) => executar(entrada).linhas.map((l) => l.texto)

describe('executar', () => {
  it('help lista os comandos locais', () => {
    const t = textos('help').join('\n')
    expect(t).toContain('help')
    expect(t).toContain('clear')
  })

  it('comando desconhecido gera erro com o nome digitado', () => {
    const { linhas } = executar('xyz argumento')
    expect(linhas).toEqual([{ tipo: 'erro', texto: 'comando não encontrado: xyz' }])
  })

  it('echo, date, whoami e about não existem mais', () => {
    for (const nome of ['echo', 'date', 'whoami', 'about']) {
      expect(textos(`${nome} x`)).toEqual([`comando não encontrado: ${nome}`])
    }
  })

  it('entrada vazia ou só de espaços não produz saída', () => {
    expect(executar('')).toEqual({ linhas: [], limpar: false })
    expect(executar('   ')).toEqual({ linhas: [], limpar: false })
  })

  it('ignora maiúsculas e espaços extras', () => {
    expect(executar('  CLEAR  ')).toEqual({ linhas: [], limpar: true })
  })

  it('clear devolve limpar: true e nenhuma linha', () => {
    expect(executar('clear')).toEqual({ linhas: [], limpar: true })
  })
})
