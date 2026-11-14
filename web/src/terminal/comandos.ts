import type { Sessao } from './api/sessao'
import type { Linha } from './linhas'

export interface Resultado {
  linhas: Linha[]
  limpar: boolean
  sessao?: Sessao
}

const saida = (texto: string): Linha => ({ tipo: 'saida', texto })

export const AJUDA_LOCAL: Array<[string, string]> = [
  ['help', 'lista os comandos disponíveis'],
  ['clear', 'limpa a tela'],
]

export function linhasAjudaLocal(): Linha[] {
  return [
    saida('Comandos disponíveis:'),
    ...AJUDA_LOCAL.map(([c, d]) => saida(`  ${c.padEnd(14)} ${d}`)),
  ]
}

export function executar(entrada: string): Resultado {
  const texto = entrada.trim()
  if (texto === '') return { linhas: [], limpar: false }

  const espaco = texto.search(/\s/)
  const nome = (espaco === -1 ? texto : texto.slice(0, espaco)).toLowerCase()

  switch (nome) {
    case 'help':
      return { linhas: linhasAjudaLocal(), limpar: false }
    case 'clear':
      return { linhas: [], limpar: true }
    default:
      return {
        linhas: [{ tipo: 'erro', texto: `comando não encontrado: ${nome}` }],
        limpar: false,
      }
  }
}
