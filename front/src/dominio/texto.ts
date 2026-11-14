/** Minúsculas, sem acentos e sem espaços nas pontas: base da busca por texto. */
export function normalizar(texto: string): string {
  return texto
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
    .trim()
}

/** Valores distintos, ordenados no padrão pt-BR. */
export function ordenarPtBr(valores: string[]): string[] {
  return [...new Set(valores)].sort((a, b) => a.localeCompare(b, 'pt-BR'))
}
