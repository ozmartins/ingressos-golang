export type TipoLinha = 'sistema' | 'entrada' | 'saida' | 'erro' | 'tabela'

export interface Linha {
  tipo: TipoLinha
  texto: string
}

export const MAX_LINHAS = 500

export function adicionarLinhas(atuais: Linha[], novas: Linha[]): Linha[] {
  const todas = [...atuais, ...novas]
  return todas.length > MAX_LINHAS ? todas.slice(todas.length - MAX_LINHAS) : todas
}
