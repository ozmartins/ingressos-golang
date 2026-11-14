import type { Filme } from '../api/tipos'
import { normalizar, ordenarPtBr } from './texto'

export interface FiltrosFilmes {
  texto: string
  genero: string
  classificacao: string
  status: '' | 'EM_CARTAZ' | 'BREVE'
}

export const filtrosFilmesVazios: FiltrosFilmes = {
  texto: '',
  genero: '',
  classificacao: '',
  status: '',
}

/** O catálogo guarda vários gêneros em um texto: "Drama, Romance, Guerra". */
export function generosDe(filme: Filme): string[] {
  return filme.genero
    .split(',')
    .map((genero) => genero.trim())
    .filter((genero) => genero !== '')
}

export function filtrarFilmes(filmes: Filme[], filtros: FiltrosFilmes): Filme[] {
  const busca = normalizar(filtros.texto)
  return filmes.filter(
    (filme) =>
      (busca === '' || normalizar(filme.titulo).includes(busca)) &&
      (filtros.genero === '' || generosDe(filme).includes(filtros.genero)) &&
      (filtros.classificacao === '' || filme.classificacao_etaria === filtros.classificacao) &&
      (filtros.status === '' || filme.status === filtros.status),
  )
}

export function opcoesFilmes(filmes: Filme[]): { generos: string[]; classificacoes: string[] } {
  return {
    generos: ordenarPtBr(filmes.flatMap(generosDe)),
    classificacoes: ordenarPtBr(filmes.map((filme) => filme.classificacao_etaria)),
  }
}
