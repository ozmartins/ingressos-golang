import type { Sessao } from './api/sessao'
import type { Dialogo } from './dialogo'
import type { Linha } from './linhas'

export interface Resultado {
  linhas: Linha[]
  limpar: boolean
  sessao?: Sessao
  /** URL para onde o navegador deve ir (login no Keycloak). */
  redirecionar?: string
  /** Pede para fechar a aba do navegador (comando exit). */
  fechar?: boolean
  /** Diálogo de preenchimento: objeto = ativo/atualizado, null = encerrado, ausente = sem mudança. */
  dialogo?: Dialogo | null
}

const saida = (texto: string): Linha => ({ tipo: 'saida', texto })

export const AJUDA_LOCAL: Array<[string, string]> = [
  ['help', 'lista os comandos disponíveis'],
  ['clear', 'limpa a tela'],
  ['exit', 'fecha a aba do navegador'],
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
    case 'exit':
      return {
        linhas: [{ tipo: 'sistema', texto: 'Fechando a aba...' }],
        limpar: false,
        fechar: true,
      }
    default:
      return {
        linhas: [{ tipo: 'erro', texto: `comando não encontrado: ${nome}` }],
        limpar: false,
      }
  }
}
