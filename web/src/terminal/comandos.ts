import type { Linha } from './linhas'

export interface Resultado {
  linhas: Linha[]
  limpar: boolean
}

const saida = (texto: string): Linha => ({ tipo: 'saida', texto })

const AJUDA: Array<[string, string]> = [
  ['help', 'lista os comandos disponíveis'],
  ['clear', 'limpa a tela'],
  ['echo <texto>', 'repete o texto informado'],
  ['date', 'mostra a data e a hora atuais'],
  ['whoami', 'mostra o usuário atual'],
  ['about', 'sobre este terminal'],
]

export function executar(entrada: string, agora: Date): Resultado {
  const texto = entrada.trim()
  if (texto === '') return { linhas: [], limpar: false }

  const espaco = texto.search(/\s/)
  const nome = (espaco === -1 ? texto : texto.slice(0, espaco)).toLowerCase()
  const argumentos = espaco === -1 ? '' : texto.slice(espaco).trim()

  switch (nome) {
    case 'help':
      return {
        linhas: [
          saida('Comandos disponíveis:'),
          ...AJUDA.map(([c, d]) => saida(`  ${c.padEnd(14)} ${d}`)),
        ],
        limpar: false,
      }
    case 'clear':
      return { linhas: [], limpar: true }
    case 'echo':
      return { linhas: [saida(argumentos)], limpar: false }
    case 'date':
      return { linhas: [saida(agora.toLocaleString('pt-BR'))], limpar: false }
    case 'whoami':
      return { linhas: [saida('visitante')], limpar: false }
    case 'about':
      return {
        linhas: [
          saida('Terminal retrô de monitor de fósforo.'),
          saida('Os comandos são simulados no navegador.'),
        ],
        limpar: false,
      }
    default:
      return {
        linhas: [{ tipo: 'erro', texto: `comando não encontrado: ${nome}` }],
        limpar: false,
      }
  }
}
